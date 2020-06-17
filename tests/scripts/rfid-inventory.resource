# Copyright (C) 2020 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# @file rfid-inventory-service.resource

*** Settings ***
| Documentation    |
| ...              | This is a Robot resource file for LRFID Inventory Service
| ...              | collection common build-type RFID Inventory with 
| ...              | meaningful Keywords.
| Library          | DateTime                                                        
| Library          | Process                                                         
| Library          | OperatingSystem 



*** Variables ***



*** Keywords ***

####=============================================================####

| Clone RFID Inventory Environment
|                 | [Arguments]                 | ${git_config}=                                                              | ${host_environment}=                                     |
|                 | ${start_time}=              | Get Time                                                                    |
|                 | ${start_time}=              | Convert Date                                                                | ${start_time}                                            | epoch |
|                 | Log to Console              | Clone RFID Inventory Environment                                            |
|                 | Should Match RegExp         | ${host_environment['working_dir']}                               | /tmp/.+  | Working Directories should be nested in the `/tmp` directory    |
|                 | ${result}= | Run Process    | rm -fr ${host_environment['working_dir']}                                                          | shell=true |
|                 | Log to Console              | ${result.stdout}                                                 | ${result.stderr}           |
|                 | ${result}= | Run Process    | mkdir -p ${host_environment['working_dir']}                                                         | shell=true |
|                 | Log to Console              | ${result.stdout}                                                 | ${result.stderr}           |
| | ${result}=    | Run Process                 | git config --global credential.helper store                                                         | shell=true | cwd=${host_environment['working_dir']} |
|                 | Log to Console              | ${result.stdout}                                                 | ${result.stderr}           |
| | ${result}=    | Run Process                 | set +x && echo "https://%{SERVICE_TOKEN}:x-oauth-basic@github.impcloud.net" > ~/.git-credentials    | shell=true |
|                 | Log to Console              | ${result.stdout}                                                 | ${result.stderr}           |
| | ${result}=    | Run Process                 | git clone ${git_config['repo']} --branch ${git_config['branch']}                                    | shell=true | cwd=${host_environment['working_dir']} |
|                 | Log to Console              | ${result.stdout}                                                 | ${result.stderr}           |
#| | ${result}=    | Run Process                 | git clone ${git_config['llrp_repo']} --branch ${git_config['llrp_branch']}                          | shell=true | cwd=${host_environment['working_dir']} |
#|                 | Log to Console              | ${result.stdout}                                                | ${result.stderr}           |


####=============================================================####

| Build RFID Inventory Environment
|                   | [Arguments]            | ${git_config}=                        | ${host_environment}=    |
|                   | ${start_time}=         | Get Time                              |
|                   | ${start_time}=         | Convert Date                          | ${start_time}           | epoch |
|                   | Log to Console         | Build RFID Inventory Environment            |
|                   | Should Match RegExp    | ${host_environment['working_dir']}    | /tmp/.+                 | Working Directories should be nested in the `/tmp` directory      |
| | ${result}=      | Run Process            | ls "-R"                               | shell=true              | cwd=/utils            |
|                   | Log to Console         | ${result.stdout}                      | ${result.stderr}        |
| | ${result}=      | Run Process            | ./installPkg.sh                       | shell=true              | cwd=/utils            |
|                   | Log to Console         | ${result.stdout}                      | ${result.stderr}        |
#| | ${result}=      | Run Process            | make "docker"                         |  shell=true             | cwd=${host_environment['working_dir']}/device-llrp-go         |
#|                   | Log to Console         | ${result.stdout}                      | ${result.stderr}        | 
| | ${result}=      | Run Process            | make "docker"                         |  shell=true             | cwd=/build            |
|                   | Log to Console         | ${result.stdout}                      | ${result.stderr}        |
| | Log To Console  | \n -----: Docker Build Inventory Env Setup is started...Take few minutes- :----\n  |
#| | ${result}=      | Run Process            | make "run"                            | shell=true  |  stdout=DEVNULL | stderr=DEVNULL  | cwd=/build     | 
#|                   | Log to Console         | ${result.stdout}                      | ${result.stderr}        |
| | ${result}=      | Run Process            | make "run"                            | shell=true  |  cwd=/build     | 
|                   | Log to Console         | ${result.stdout}                      | ${result.stderr}        |
| | Log To Console  | \n -----: RFID Inventory Service Instances - Docker Containers :----\n  |
| | ${result}=      | Run Process            | docker "ps" "--format" "{{.Names}}"   | shell=true              | cwd=/build     |
|                   | Log to Console         |  ${result.stdout}                     | ${result.stderr}        |
|                   | Log to Console         |  \n RFID Inventory Service Build Done.....\n     |
|                   | Log to Console         |  \n After Build wait for 10s.....\n   |

####=============================================================####

| Simulator RFID Inventory Environment
|                 | [Arguments]            | ${git_config}=                         | ${host_environment}=    |
|                 | ${start_time}=         | Get Time                               |
|                 | ${start_time}=         | Convert Date                           | ${start_time}           | epoch |
|                 | Log to Console         | Simulator RFID Inventory Environment         |
|                 | Log to Console         | \n RFID Inventory Service Simulating......\n      |
|                 | Should Match RegExp    | ${host_environment['working_dir']}     | /tmp/.+                 | Working Directories should be nested in the `/tmp` directory        |
#| | ${result}=    | Run Process            | docker ps --format  {{.Names}}        | shell=true              | cwd=${host_environment['working_dir']}     |
#|                 | Log to Console         | ${result.stdout}                      | ${result.stderr}        |


####=============================================================####

| Shutdown RFID Inventory Simulator Environment
|                 | [Arguments]            | ${git_config}=                         | ${host_environment}=    |
|                 | ${start_time}=         | Get Time                               |
|                 | ${start_time}=         | Convert Date                           | ${start_time}           | epoch |
|                 | Log to Console         | Shutdown RFID Inventory Service Simulator Environment |


####=============================================================####

| Shutdown RFID Inventory Environment
|                 | [Arguments]            | ${git_config}=                              | ${host_environment}=    |
|                 | ${start_time}=         | Get Time                                    | 
|                 | ${start_time}=         | Convert Date                                | ${start_time}           | epoch |
|                 | Log to Console         | Shutdown RFID Inventory Environment...            |
|                 | Should Match RegExp    | ${host_environment['working_dir']}          | /tmp/.+                 | Working Directories should be nested in the `/tmp` directory    |
| | ${result}=    | Run Process            | make "stop"                                 | shell=true              | cwd=/build    |
|                 | Log to Console         | ${result.stdout}                            | ${result.stderr}        |
| | ${result}=    | Run Process            | make "down"                                   | shell=true            | cwd=/build    |
|                 | Log to Console         | ${result.stdout}                            | ${result.stderr}        |
| | ${result}=    | Run Process            | ./rmvDockerImgs.sh                          | shell=true              | cwd=/utils    |
|                 | Log to Console         | ${result.stdout}                            | ${result.stderr}        |
|                 | Log to Console          |   NOTE:: ALL Shutdown Completed            |
#| | ${result}=    | Run Process            | sduo rm -rf rfid-inventory-test-platform   | shell=true              | cwd=${host_environment['working_dir']} |
#|                 | Log to Console         | ${result.stdout}                           | ${result.stderr}        |
#| | ${result}=    | Run Process            | sudo docker network prune -f               | shell=true              | cwd=${host_environment['working_dir']} |
#|                 | Log to Console         | ${result.stdout}                           | ${result.stderr}        |
#| | ${result}=    | Run Process            | docker volume ls -f dangling=true          | shell=true              | cwd=${host_environment['working_dir']} |
#|                 | Log to Console         | ${result.stdout}                           | ${result.stderr}        |
