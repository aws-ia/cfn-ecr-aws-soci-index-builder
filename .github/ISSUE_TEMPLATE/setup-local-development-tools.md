---
name: Setup Local Development tools
about: Setup Local Development tools (taskcat)
title: Setup Local Development tools
labels: ''
assignees: ''

---

### Install testing/validation tools
Prior to raising PR ,test and validate your code using `taskcat`. This is an optional step but highly recommended. Following these step will save to time and ensure that you PR is merged promptly. 

> `taskcat` requires **python3** 

1. Install `taskcat` via pip
` pip install taskcat`

2. Create `taskcat.yaml` Follow documentation [here](https://aws-ia.github.io/taskcat/docs/usage/GENERAL_USAGE.html#config-files) 
> A sample .taskcat.yml can be found in your repo

`taskcat` requires access to an AWS account, this can be done by any of the following mechanisms:

Environment variables `AWS_ACCESS_KEY_ID` `AWS_SECRET_ACCESS_KEY` `AWS_SESSION_TOKEN`
Shared credential file `(~/.aws/credentials)`
AWS config file `(~/.aws/config)`
Boto2 config file `(/etc/boto.cfg and ~/.boto)`
(https://boto3.amazonaws.com/v1/documentation/api/latest/guide/configuration.html).

Go to [taskcat.io](https://taskcat.io/) for full documentation 

3. Run a local test this will produce a pass fail report. Once all test are passing proceed to the next step
<img width="1286" alt="Screen Shot 2022-01-12 at 3 02 15 PM" src="https://user-images.githubusercontent.com/5912128/149236817-fa8ddb51-35aa-42ba-932c-741b2b40a656.png">

- [ ] Commit `taskcat` configs to PR branch

5. Close this issue 

6.  Raise a PR to merge to main
- [ ] Add taskcat test results screenshots to PR comments
