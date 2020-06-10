# @file rfid-inventory-install.robot

# Copyright (C) 2020 Intel Corporation
# SPDX-License-Identifier: Apache-2.0


*** Settings ***
| Documentation     | This file collects the Test Cases associated with the
| ...               | RFID Inventory Service Installation, and generally all Test Cases will
| ...               | be prefixed with their corresponding Test Case(TC) Number in Rally.
| Resource          | /scripts/rfid-inventory.resource                                                         |                                                  
| Library           | /scripts/install_check/container_count.py                                                |
| Suite Setup       | Suite_Setup                                                                              |
| Suite Teardown    | Suite_Teardown                                                                           |
| Test Setup        | Common_Setup                                                                             |
| Test Teardown     | Common_Teardown                                                                          |
| Variables         | /config/config.yaml                                                                      |

*** Variables ***


*** Keywords ***
| Suite_Setup
|    | Set Log Level                                                                        | DEBUG                                        |
|    | Log To Console                                                                       | ${SUITE NAME}: Suite Setup started.          |
|    | rfid-inventory.Clone RFID Inventory Environment   | host_environment=${execution_environment}    | git_config=${git_config}                     |
|    | rfid-inventory.Build RFID Inventory Environment                                                  | host_environment=${execution_environment}    |
|    | Sleep | 10s                                                                          |
#|    | rfid-inventory.Simulator RFID Inventory Environment                                       | host_environment=${execution_environment}    |
#|    | Sleep | 10s                                                                         |
|    | Log To Console                                                                       | ${SUITE NAME}: Suite Setup finished.         |


| Common_Setup
|    | Log To Console    | \n${SUITE NAME}: Common Setup started.                    |   
|    | Log To Console    | ${SUITE NAME}: Common Setup finished.                     |   

| Common_Teardown
|    | Log To Console    | \n${SUITE NAME}: Common Teardown started.                 |  
|    | Log To Console    | ${SUITE NAME}: Common Teardown finished.                  |

| Suite_Teardown
|    | Log To Console                  | ${SUITE NAME}: Suite Teardown started.       |
#|    | rfid-inventory.Shutdown RFID Inventory Environment    | host_environment=${execution_environment}    |
|    | rfid-inventory.Shutdown RFID Inventory Environment              | host_environment=${execution_environment}    |
|    | Log To Console                  | ${SUITE NAME}: Suite Teardown finished.      |

*** Test cases ***
| TC0001_Execute_RFID Inventory_Service_Test
|    | [Tags]             | Run                                       | generic | install
|    | [Documentation]    | RFID Inventory Installtion Check          |
|    | Log To Console     | Testcase ${TEST NAME} started.            |
|    | ${status}=         | verify_containers_running | positive      |
|    | Log To Console     | Status = ${status}                        |
|    | Should Be True     | ${status}                                 |
|    | Log To Console     | Testcase ${TEST NAME} finished.           |
