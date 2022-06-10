package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	apimodels "github.com/keptn/go-utils/pkg/api/models"
	keptncommon "github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	"github.com/keptn/keptn/shipyard-controller/common"
	"github.com/keptn/keptn/shipyard-controller/handler/fake"
	"github.com/keptn/keptn/shipyard-controller/models"
	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"github.com/tryvium-travels/memongo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	fakeclient "k8s.io/client-go/testing"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

const natsTestPort = 8369
const mongoDBVersion = "4.4.9"

const testShipyardFile = `apiVersion: spec.keptn.sh/0.2.0
kind: Shipyard
metadata:
  name: test-shipyard
spec:
  stages:
  - name: dev
    sequences:
    - name: delivery
      tasks:
      - name: mytask
        properties:  
          strategy: direct
      - name: test
        properties:
          kind: functional
      - name: evaluation 
      - name: release 
    - name: rollback
      tasks:
      - name: rollback
      triggeredOn:
        - event: dev.artifact-delivery.finished
          selector:
            match:
              result: fail
  - name: hardening
    sequences:
    - name: artifact-delivery
      triggeredOn:
        - event: dev.artifact-delivery.finished
      tasks:
      - name: deployment
        properties: 
          strategy: blue_green_service
      - name: test
        properties:  
          kind: performance
      - name: evaluation
      - name: release

  - name: production
    sequences:
    - name: artifact-delivery 
      triggeredOn:
        - event: hardening.artifact-delivery.finished
      tasks:
      - name: deployment
        properties:
          strategy: blue_green
      - name: release
      
    - name: remediation
      tasks:
      - name: remediation
      - name: evaluation`

func setupNatsServer(port int) *server.Server {
	opts := natsserver.DefaultTestOptions
	opts.Port = port
	opts.JetStream = true
	svr := natsserver.RunServer(&opts)
	return svr
}

func setupLocalMongoDB() func() {
	mongoServer, err := memongo.Start(mongoDBVersion)
	randomDbName := memongo.RandomDatabase()

	os.Setenv("MONGODB_DATABASE", randomDbName)
	os.Setenv("MONGODB_EXTERNAL_CONNECTION_STRING", fmt.Sprintf("%s/%s", mongoServer.URI(), randomDbName))

	var mongoDBClient *mongo.Client
	mongoDBClient, err = mongo.NewClient(options.Client().ApplyURI(mongoServer.URI()))
	if err != nil {
		log.Fatalf("Mongo Client setup failed: %s", err)
	}
	err = mongoDBClient.Connect(context.TODO())
	if err != nil {
		log.Fatalf("Mongo Server setup failed: %s", err)
	}

	return func() { mongoServer.Stop() }
}

func TestMain(m *testing.M) {
	natsServer := setupNatsServer(natsTestPort)
	defer natsServer.Shutdown()
	defer setupLocalMongoDB()()
	m.Run()
}

func Test_getDurationFromEnvVar(t *testing.T) {
	type args struct {
		envVarValue string
	}
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{
			name: "get default value",
			args: args{
				envVarValue: "",
			},
			want: 432000 * time.Second,
		},
		{
			name: "get configured value",
			args: args{
				envVarValue: "10s",
			},
			want: 10 * time.Second,
		},
		{
			name: "get configured value",
			args: args{
				envVarValue: "2m",
			},
			want: 120 * time.Second,
		},
		{
			name: "get configured value",
			args: args{
				envVarValue: "1h30m",
			},
			want: 5400 * time.Second,
		},
		{
			name: "get default value because of invalid config",
			args: args{
				envVarValue: "invalid",
			},
			want: 432000 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("LOG_TTL", tt.args.envVarValue)
			if got := getDurationFromEnvVar("LOG_TTL", envVarLogsTTLDefault); got != tt.want {
				t.Errorf("getLogTTLDurationInSeconds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_LeaderElection(t *testing.T) {
	var (
		onNewLeader = make(chan struct{})
		onRelease   = make(chan struct{})
		lockObj     runtime.Object
	)
	c := &fakek8s.Clientset{}

	shipyard := &fake.IShipyardControllerMock{
		StartDispatchersFunc: func(ctx context.Context, mode common.SDMode) {
			time.After(5 * time.Second)
			close(onNewLeader)
		},
		StopDispatchersFunc: func() {
			onNewLeader = make(chan struct{})
			close(onRelease)
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	// create lock
	c.AddReactor("create", "leases", func(action fakeclient.Action) (handled bool, ret runtime.Object, err error) {
		lockObj = action.(fakeclient.CreateAction).GetObject()
		return true, lockObj, nil
	})

	//fail with no lock
	c.AddReactor("get", "leases", func(action fakeclient.Action) (handled bool, ret runtime.Object, err error) {
		if lockObj != nil {
			return true, lockObj, nil
		}
		return true, nil, errors.NewNotFound(action.(fakeclient.GetAction).GetResource().GroupResource(), action.(fakeclient.GetAction).GetName())
	})

	c.AddReactor("update", "leases", func(action fakeclient.Action) (handled bool, ret runtime.Object, err error) {
		// Second update (first renew) should return our canceled error
		// FakeClient doesn't do anything with the context so we're doing this ourselves

		lockObj = action.(fakeclient.UpdateAction).GetObject()
		return true, lockObj, nil

	})

	c.AddReactor("*", "*", func(action fakeclient.Action) (bool, runtime.Object, error) {
		t.Errorf("unreachable action. testclient called too many times: %+v", action)
		return true, nil, fmt.Errorf("unreachable action")
	})

	newReplica := func() {
		LeaderElection(c.CoordinationV1(), ctx, shipyard.StartDispatchersFunc, shipyard.StopDispatchers)
	}
	go newReplica()

	// Wait for one replica to become the leader
	select {
	case <-onNewLeader:
		// stopping the leader

		go newReplica() // leader already there one of the two may fail but not panic
		cancel()
		select {
		case <-onRelease:
			//reset chan for next leader
			onRelease = make(chan struct{})
		case <-time.After(10 * time.Second):
			t.Fatal("failed to release lock")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("failed to become the leader")
	}

}

func Test__main(t *testing.T) {
	fakeClient := fakek8s.NewSimpleClientset()

	// TODO set up fake http server for configuration service - for retrieving shipyard.yaml

	fakeCS, closeFunc := setupFakeConfigurationService()
	defer closeFunc()

	natsURL := fmt.Sprintf("nats://127.0.0.1:%d", natsTestPort)
	// TODO set up env vars
	os.Setenv(envVarConfigurationSvcEndpoint, fakeCS.URL)
	os.Setenv(envVarNatsURL, natsURL)
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv(envVarEventDispatchIntervalSec, "1")
	os.Setenv(envVarSequenceDispatchIntervalSec, "1s")

	natsClient, err := newTestNatsClient(natsURL, t)

	require.Nil(t, err)

	// TODO insert a project into the MV

	go _main(fakeClient)

	projectName := "my-project"
	serviceName := "my-service"
	shipyardContent := base64.StdEncoding.EncodeToString([]byte(testShipyardFile))

	createProject := models.CreateProjectParams{
		Name:     &projectName,
		Shipyard: &shipyardContent,
	}

	marshal, err := json.Marshal(createProject)

	require.Nil(t, err)

	c := http.Client{
		Timeout: 2 * time.Second,
	}

	require.Eventually(t, func() bool {
		req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/v1/project", bytes.NewBuffer(marshal))
		if err != nil {
			return false
		}

		resp, err := c.Do(req)

		if err != nil {
			return false
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return false
		}
		return true
	}, 10*time.Second, 100*time.Millisecond)

	service := models.CreateServiceParams{
		ServiceName: &serviceName,
	}

	marshal, err = json.Marshal(service)
	require.Nil(t, err)

	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/v1/project/"+projectName+"/service", bytes.NewBuffer(marshal))
	require.Nil(t, err)

	resp, err := c.Do(req)

	require.Nil(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	// TODO now we should be able to trigger sequences
	source := "golang-test"

	context := natsClient.triggerSequence(projectName, serviceName, "dev", "delivery")

	t.Logf("wait for the sequence state to be available")
	verifySequenceEndsUpInState(t, projectName, context, 2*time.Minute, []string{apimodels.SequenceStartedState})
	t.Log("received the expected state!")

	t.Logf("trigger a second sequence - this one should stay in 'waiting' state until the previous sequence is finished")
	secondContext := natsClient.triggerSequence(projectName, serviceName, "dev", "delivery")

	t.Logf("checking if the second sequence is in state 'waiting'")
	verifySequenceEndsUpInState(t, projectName, secondContext, 2*time.Minute, []string{apimodels.SequenceWaitingState})
	t.Log("received the expected state!")

	t.Logf("check if mytask.triggered has been sent for first sequence - this one should be available")
	triggeredEventOfFirstSequence := natsClient.getLatestEventOfType(*context.KeptnContext, projectName, "dev", keptnv2.GetTriggeredEventType("mytask"))
	require.NotNil(t, triggeredEventOfFirstSequence)

	t.Logf("check if mytask.triggered has been sent for second sequence - this one should NOT be available")
	triggeredEventOfSecondSequence := natsClient.getLatestEventOfType(*secondContext.KeptnContext, projectName, "dev", keptnv2.GetTriggeredEventType("mytask"))
	require.Nil(t, triggeredEventOfSecondSequence)

	t.Logf("send .started and .finished event for task of first sequence")
	cloudEvent := keptnv2.ToCloudEvent(*triggeredEventOfFirstSequence)

	keptn, err := keptnv2.NewKeptn(&cloudEvent, keptncommon.KeptnOpts{EventSender: natsClient})
	require.Nil(t, err)

	t.Logf("send started event")
	_, err = keptn.SendTaskStartedEvent(nil, source)
	require.Nil(t, err)

	t.Logf("send finished event with result=fail")
	_, err = keptn.SendTaskFinishedEvent(&keptnv2.EventData{
		Status: keptnv2.StatusSucceeded,
		Result: keptnv2.ResultFailed,
	}, source)
	require.Nil(t, err)

	t.Logf("now that all tasks for the first sequence have been executed, the second sequence should eventually have the status 'started'")
	t.Logf("waiting for state with keptnContext %s to have the status %s", *context.KeptnContext, apimodels.SequenceStartedState)
	verifySequenceEndsUpInState(t, projectName, secondContext, 2*time.Minute, []string{apimodels.SequenceStartedState})
	t.Log("received the expected state!")

	t.Logf("check if mytask.triggered has been sent for second sequence - now it should be available")
	require.Eventually(t, func() bool {
		triggeredEventOfSecondSequence = natsClient.getLatestEventOfType(*secondContext.KeptnContext, projectName, "dev", keptnv2.GetTriggeredEventType("mytask"))
		return triggeredEventOfSecondSequence != nil
	}, 2*time.Second, 100*time.Millisecond)

}

func setupFakeConfigurationService() (*httptest.Server, func()) {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		if strings.Contains(r.RequestURI, "/shipyard.yaml") {
			w.WriteHeader(200)
			encodedShipyard := base64.StdEncoding.EncodeToString([]byte(testShipyardFile))
			res := apimodels.Resource{
				ResourceContent: encodedShipyard,
			}

			marshal, _ := json.Marshal(res)
			w.Write(marshal)

			return
		} else if strings.Contains(r.RequestURI, "/metadata.yaml") {
			res := apimodels.Resource{
				Metadata: &apimodels.Version{
					Version: "my-commit-id",
				},
			}

			marshal, _ := json.Marshal(res)
			w.Write(marshal)

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))

	ts.Config.Addr = "127.0.0.1"
	ts.Start()

	return ts, ts.Close
}

type testNatsClient struct {
	*nats.Conn
	t              *testing.T
	receivedEvents []apimodels.KeptnContextExtendedCE
	sync.RWMutex
}

func newTestNatsClient(natsURL string, t *testing.T) (*testNatsClient, error) {
	natsConn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}

	tnc := &testNatsClient{
		t:    t,
		Conn: natsConn,
	}

	_, err = tnc.Subscribe("sh.keptn.>", func(msg *nats.Msg) {
		tnc.onEvent(msg)
	})
	if err != nil {
		return nil, err
	}

	return tnc, nil
}

func (n *testNatsClient) onEvent(msg *nats.Msg) {
	n.Lock()
	defer n.Unlock()

	n.t.Logf("Received event of type: %s", msg.Subject)
	ev := &apimodels.KeptnContextExtendedCE{}

	if err := json.Unmarshal(msg.Data, ev); err == nil {
		n.receivedEvents = append(n.receivedEvents, *ev)
	}
}

func (n *testNatsClient) triggerSequence(projectName, serviceName, stageName, sequenceName string) *apimodels.EventContext {
	source := "golang-test"
	eventType := keptnv2.GetTriggeredEventType(stageName + "." + sequenceName)
	n.t.Log("triggering task sequence")

	keptnContext := uuid.NewString()

	eventPayload := apimodels.KeptnContextExtendedCE{
		Contenttype: "application/json",
		Data: keptnv2.DeploymentTriggeredEventData{
			EventData: keptnv2.EventData{
				Project: projectName,
				Stage:   stageName,
				Service: serviceName,
				Result:  keptnv2.ResultPass,
			},
		},
		ID:                 uuid.NewString(),
		Shkeptncontext:     keptnContext,
		Shkeptnspecversion: "0.2.0",
		Source:             &source,
		Specversion:        "1.0",
		Type:               &eventType,
	}

	marshal, err := json.Marshal(eventPayload)
	require.Nil(n.t, err)

	err = n.Publish(eventType, marshal)

	return &apimodels.EventContext{
		KeptnContext: &keptnContext,
	}
}

func (n *testNatsClient) SendEvent(event cloudevents.Event) error {
	m, _ := json.Marshal(event)
	return n.Publish(event.Type(), m)
}

func (n *testNatsClient) Send(ctx context.Context, event cloudevents.Event) error {
	return n.SendEvent(event)
}

func (n *testNatsClient) getLatestEventOfType(keptnContext, projectName, stage, eventType string) *apimodels.KeptnContextExtendedCE {
	for _, ev := range n.receivedEvents {
		if ev.Shkeptncontext == keptnContext && *ev.Type == eventType {
			ed := &keptnv2.EventData{}
			err := keptnv2.Decode(ev.Data, ed)
			require.Nil(n.t, err)
			if ed.Project == projectName && ed.Stage == stage {
				return &ev
			}
		}
	}
	return nil
}

func verifySequenceEndsUpInState(t *testing.T, projectName string, context *apimodels.EventContext, timeout time.Duration, desiredStates []string) {
	t.Logf("waiting for state with keptnContext %s to have the status %s", *context.KeptnContext, desiredStates)
	c := http.Client{}

	require.Eventually(t, func() bool {
		req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/v1/sequence/"+projectName+"?keptnContext="+*context.KeptnContext, nil)
		if err != nil {
			return false
		}

		resp, err := c.Do(req)
		if err != nil {
			return false
		}

		respBytes, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return false
		}

		states := &apimodels.SequenceStates{}

		err = json.Unmarshal(respBytes, states)
		if err != nil {
			return false
		}

		for _, state := range states.States {
			if doesSequenceHaveOneOfTheDesiredStates(state, context, desiredStates) {
				return true
			}
		}
		return false
	}, timeout, 100*time.Millisecond)
}

func doesSequenceHaveOneOfTheDesiredStates(state apimodels.SequenceState, context *apimodels.EventContext, desiredStates []string) bool {
	if state.Shkeptncontext == *context.KeptnContext {
		for _, desiredState := range desiredStates {
			if state.State == desiredState {
				return true
			}
		}
	}
	return false
}
