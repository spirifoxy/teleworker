Feature: stop the job
    In order to finish running command
    As an end user
    I need to stop the job

    Scenario: should stop the job
    Given I pass my command sleep
    And I pass command argument 10
    And the job was created
    When I try to stop the job
    Then the response is success

    Scenario: should fail to stop unexistent job
    Given I pass my command echo
    And I pass command argument 1
    And the job was created
    When I try to stop some random job
    Then the response is error
