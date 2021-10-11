Feature: status of the job
    In order get state of the running command
    As an end user
    I need to query status of the job

    Scenario: should get status of the alive job
    Given I pass my command sleep
    And I pass command argument 10
    And the job was created
    When I try to get status of the job
    Then the response is success
    And I see the job is still running

    Scenario: should get status of the finished job
    Given I pass my command echo
    And I pass command argument 1
    And the job was created
    And I wait for a second
    When I try to get status of the job
    Then the response is success
    And I see the job is finished
