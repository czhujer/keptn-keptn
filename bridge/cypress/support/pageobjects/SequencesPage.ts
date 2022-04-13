export class SequencesPage {
  private readonly sequenceWaitingMessage = ' Sequence is waiting for previous sequences to finish. ';

  public visit(projectName: string): this {
    cy.visit(`/project/${projectName}/sequence`);
    return this;
  }

  public interceptRemediationSequences(): this {
    cy.intercept('/api/v1/metadata', { fixture: 'metadata.mock' });
    cy.intercept('/api/bridgeInfo', { fixture: 'bridgeInfo.mock' });
    cy.intercept('/api/project/sockshop?approval=true&remediation=true', {
      fixture: 'get.project.sockshop.remediation.mock',
    });
    cy.intercept('/api/hasUnreadUniformRegistrationLogs', { body: false });

    cy.intercept('/api/controlPlane/v1/project?disableUpstreamSync=true&pageSize=50', { fixture: 'projects.mock' });
    cy.intercept('/api/controlPlane/v1/sequence/sockshop?pageSize=25', {
      fixture: 'get.sequences.remediation.mock',
    }).as('Sequences');
    cy.intercept('/api/controlPlane/v1/sequence/sockshop?pageSize=25&fromTime=*', {
      body: {
        states: [],
      },
    });
    cy.intercept('/api/project/sockshop/sequences/metadata', { fixture: 'sequence.metadata.mock' }).as(
      'SequencesMetadata'
    );

    cy.intercept('/api/mongodb-datastore/event?keptnContext=cfaadbb1-3c47-46e5-a230-2e312cf1828a&project=sockshop', {
      fixture: 'get.events.cfaadbb1-3c47-46e5-a230-2e312cf1828a.mock.json',
    });
    cy.intercept('/api/mongodb-datastore/event?keptnContext=cfaadbb1-3c47-46e5-a230-d0f055f4f518&project=sockshop', {
      fixture: 'get.events.cfaadbb1-3c47-46e5-a230-d0f055f4f518.mock.json',
    });
    cy.intercept('/api/mongodb-datastore/event?keptnContext=29355a07-7b65-47fa-896e-06f656283c5d&project=sockshop', {
      fixture: 'get.events.29355a07-7b65-47fa-896e-06f656283c5d.mock.json',
    });

    return this;
  }

  public selectSequence(keptnContext: string): this {
    cy.byTestId(`keptn-root-events-list-${keptnContext}`).click();
    return this;
  }

  private setFilterForGroup(filterGroup: string, itemName: string, status: boolean): this {
    cy.byTestId('keptn-sequence-view-filter').find('dt-quick-filter').dtQuickFilterCheck(filterGroup, itemName, status);
    return this;
  }

  public checkServiceFilter(serviceName: string, status = true): this {
    return this.setFilterForGroup('Service', serviceName, status);
  }

  public checkStageFilter(stageName: string, status = true): this {
    return this.setFilterForGroup('Stage', stageName, status);
  }

  public checkSequenceFilter(sequenceName: string, status = true): this {
    return this.setFilterForGroup('Sequence', sequenceName, status);
  }

  public checkStatusFilter(statusName: string, status = true): this {
    return this.setFilterForGroup('Status', statusName, status);
  }

  public clearFilter(): this {
    cy.byTestId('keptn-sequence-view-filter').find('dt-quick-filter').clearDtFilter();
    return this;
  }

  public assertSequenceCount(count: number): this {
    cy.byTestId('keptn-sequence-view-roots').get('ktb-selectable-tile').should('have.length', count);
    return this;
  }

  public assertServiceNameOfSequences(serviceName: string): this {
    return this.assertSequenceTile('keptn-sequence-info-serviceName', serviceName);
  }

  public assertStageNameOfSequences(stageName: string): this {
    return this.assertStageNamesOfSequences([stageName], false);
  }

  public assertStageNamesOfSequences(stageNames: string[], validateLength = true): this {
    cy.byTestId('keptn-sequence-info-stageDetails').each((el) => {
      for (const stageName of stageNames) {
        cy.wrap(el).find('ktb-stage-badge').contains(stageName).should('exist');
      }
      if (validateLength) {
        cy.wrap(el).find('ktb-stage-badge').should('have.length', stageNames.length);
      }
    });
    return this;
  }

  public assertSequenceNameOfSequences(sequenceName: string): this {
    return this.assertSequenceTile('keptn-sequence-info-sequenceName', sequenceName);
  }

  public assertStatusOfSequences(status: string): this {
    return this.assertSequenceTile('keptn-sequence-info-status', status);
  }

  private assertSequenceTile(testId: string, expectedText: string): this {
    cy.byTestId(testId).each((el) => {
      cy.wrap(el).should('have.text', expectedText);
    });
    return this;
  }

  public assertNoSequencesMessageExists(status: boolean): this {
    cy.byTestId('keptn-noSequencesFiltered').should(status ? 'exist' : 'not.exist');
    return this;
  }

  public assertNoSequencesFilteredMessageExists(status: boolean): this {
    cy.byTestId('keptn-noSequencesFiltered').should(status ? 'exist' : 'not.exist');
    return this;
  }

  public assertLoadingOldSequencesButtonExists(status: boolean): this {
    cy.byTestId('keptn-loadingOldSequences').should(status ? 'exist' : 'not.exist');
    return this;
  }

  public assertIsLoadingSequences(status: boolean): this {
    cy.byTestId('keptn-loadingSequences').should(status ? 'exist' : 'not.exist');
    return this;
  }

  public assertIsWaitingSequence(keptnContext: string, status: boolean): this {
    cy.byTestId(`keptn-root-events-list-${keptnContext}`)
      .find('.ktb-selectable-tile-content')
      .should(status ? 'have.text' : 'not.have.text', this.sequenceWaitingMessage);
    return this;
  }

  public assertIsSelectedSequenceWaiting(status: boolean): this {
    cy.byTestId('keptn-sequence-view-sequenceDetails')
      .find('dt-alert')
      .should(status ? 'have.text' : 'not.have.text', this.sequenceWaitingMessage);
    return this;
  }

  public assertTaskFailed(taskName: string, isFailed: boolean): this {
    cy.byTestId(`keptn-task-item-${taskName}`)
      .find('ktb-expandable-tile')
      .eq(0)
      .should(isFailed ? 'have.class' : 'not.have.class', 'ktb-tile-error');
    return this;
  }

  public assertTaskSuccessful(taskName: string, isSuccess: boolean): this {
    cy.byTestId(`keptn-task-item-${taskName}`)
      .find('ktb-expandable-tile')
      .eq(0)
      .should(isSuccess ? 'have.class' : 'not.have.class', 'ktb-tile-success');
    return this;
  }

  public assertTimelineTime(stage: string, time: string): this {
    cy.get('.stage-info')
      .contains(stage)
      .parentsUntilTestId(`keptn-sequence-timeline-stage-${stage}`)
      .should('contain.text', time);
    return this;
  }

  public assertTimelineTimeLoading(stage: string, exists: boolean): this {
    cy.get('.stage-info')
      .contains(stage)
      .parentsUntilTestId(`keptn-sequence-timeline-stage-${stage}`)
      .find('dt-loading-spinner')
      .should(exists ? 'exist' : 'not.exist');
    return this;
  }

  public assertServiceName(name: string, tag?: string): this {
    const serviceName = tag ? `${name}:${tag}` : name;
    cy.byTestId('keptn-sequence-view-serviceName').should('have.text', serviceName);
    return this;
  }
}