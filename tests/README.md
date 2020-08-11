# FAST - Framework for Simplified Testing

## Layout for test infrastructure for RFID Inventory Service.


## How do I build locally in my repo?

# 1. tests/.env 
NOTE: Do not push this file to your repo/branch.
    
For tests locally, Create a file ".env" and add below these two lines -

    SERVICE_TOKEN=<Place your git token here>
    GIT_BRANCH=<Your local branch>

    e.g : `FAST` is branch name, then GIT_BRANCH=FAST

    `docker-compose config` - command to check the all env and args.


# 2. Build Command 

  At Repo branch path "rfid-inventory-service/"

  `docker-compose -f ./tests/docker-compose.yml up --build`


## How do I verify the build result/reports?

  For the test results open `reports.html` in `/tests/reports`.
  Console logs also.



## Remotely
# How do I build remotely at jenkins agent?
 
Build: `https://rrpdevops01.amr.corp.intel.com/job/RSP-Inventory-Suite/job/rfid-inventory-service/`
