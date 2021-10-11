Feature: start new job
    In order to launch my command
    As an end user
    I need to start new job

    Scenario: should create new job
    When I pass my command echo
    And I pass command argument 1
    And I try to create new job
    Then the response is success
    And I get the job uuid

    Scenario: should run complicated command
    When I pass my command bash
    And I pass command argument --
    And I pass command argument -c
    And I pass command argument cat /proc/cpuinfo | egrep '^model name' | uniq
    And I try to create new job
    Then the response is success
    And I get the job uuid