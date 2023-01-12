# Getting Started

## 1. Fork this repo 
**[IMPORTANT]: Write access is not provided to this repo. Please be sure to work in your fork**

Use fork button in the GitHub UI 

<img width="151" alt="Screen Shot 2022-07-14 at 9 34 41 AM" src="https://user-images.githubusercontent.com/5912128/179034198-b4c258a0-15e3-4ef6-9e2d-fc18a70f1dbe.png">

## 2. Clone your repo and develop the asset

**[From your local terminal]** Run `git clone <repo url>`

__Note: Repo url is within your git user namespace so you will have write access to it__

Add your code to /template/`name_of_cfntemplate.yaml`

## 3. Setting up testing toolkit (recommended)
3a. Setup a python env (3.8.x <3.9)

3b. Setup a default aws profile

```
Example of 2b.: more info here https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html
[default]
aws_access_key_id=AKIAIOSFODNN7EXAMPLE
aws_secret_access_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

3c. Install taskcat

**[From your local terminal]** Run `pip3 install taskcat`

__Note: For more info on taskcat see http://taskcat.io__

3d. Test your code

__Note: Template name much match in taskcat.yml__

```
Example: .taskcat.yml (a sample is in the root of the repo)
tests:
  nameoftest:
    parameters:
      Param1: 'Inputs to Stack'
    regions: ## Regions to test in
    - us-east-1
    template: templates/name_of_cfntemplate.yaml <# Change to the name of template
```

**[From your local terminal]** Run `taskcat test run`

```
Example Output:
 _            _             _
| |_ __ _ ___| | _____ __ _| |_
| __/ _` / __| |/ / __/ _` | __|
| || (_| \__ \   < (_| (_| | |_
 \__\__,_|___/_|\_\___\__,_|\__|


version 0.9.30
[INFO   ] : Linting passed for file: /Users/tonynv/work/github/aws-quickstart/cfn-template/templates/sample-yaml.template
[S3: -> ] s3://tcat-cfn-template-553r8x5x/cfn-template/LICENSE
[S3: -> ] s3://tcat-cfn-template-553r8x5x/cfn-template/CODEOWNERS
[S3: -> ] s3://tcat-cfn-template-553r8x5x/cfn-template/NOTICE.txt
[S3: -> ] s3://tcat-cfn-template-553r8x5x/cfn-template/scripts/scripts_userdata.sh
[S3: -> ] s3://tcat-cfn-template-553r8x5x/cfn-template/templates/sample-yaml.template
[INFO   ] : ┏ stack Ⓜ tCaT-cfn-template-sample-304f89e2a420412e842b8d9fb66e102e
[INFO   ] : ┣ region: us-east-1
[INFO   ] : ┗ status: CREATE_COMPLETE
[INFO   ] : Reporting on arn:aws:cloudformation:us-east-1:************:stack/tCaT-cfn-template-sample-304f89e2a420412e842b8d9fb66e102e/6327faf0-038d-11ed-86d3-0ae8b573018d 
[INFO   ] : Deleting stack: arn:aws:cloudformation:us-east-1:************:stack/tCaT-cfn-template-sample-304f89e2a420412e842b8d9fb66e102e/6327faf0-038d-11ed-86d3-0ae8b573018d 
[INFO   ] : ┏ stack Ⓜ tCaT-cfn-template-sample-304f89e2a420412e842b8d9fb66e102e
[INFO   ] : ┣ region: us-east-1                                                                                         
[INFO   ] : ┗ status: DELETE_COMPLETE


## Step4: Submit for Publication

(Raise a PR to the upstream repo from your fork)

When a Pull Request is raised against the upstream repo. 
- Automated tests will run and provide you instant feedback. 
When all Check have PR Check have PASSED and functional test will begin (code in all regions specfied in .taskcat.yml)

If these tests pass the PR will be merged to main and synced to S3



