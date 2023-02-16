# Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

import json
import os
import fnmatch, re
import boto3

lambda_client = boto3.client('lambda')

def lambda_handler(event, context):
    """
    Handler - Meant to be invoked by EventBridge for ECR Image Action events.

    sample event data from EventBridge:
        {
            "version": "0",
            "id": "999cccaa-eaaa-0000-1111-123456789012",
            "detail-type": "ECR Image Action",
            "source": "aws.ecr",
            "account": "123456789012",
            "time": "2016-12-16T20:43:05Z",
            "region": "us-east-1",
            "resources": [],
            "detail": {
                "result": "SUCCESS",
                "repository-name": "repository_name",
                "image-digest": "sha256:978f5f8049d3d0de30a7fc3892aafdfb323451bf682170d99154230ddefbe91e",
                "action-type": "PUSH",
                "image-tag": "hello-world"
            }
        }
    """

    try:
        repository_name = event['detail']['repository-name']
        image_tag = event['detail']['image-tag']
        image_digest = event['detail']['image-digest']

        event_source = event['source']
        detail_type = event['detail-type']
    except:
        return log_and_generate_response(400, "Invalid event")

    if event_source != "aws.ecr" or detail_type != "ECR Image Action":
        return log_and_generate_response(400, "Wrong event source or detail type")

    if repository_name == "" or image_digest == "":
        return log_and_generate_response(400, "repository name and image digest cannot be empty")

    try:
        soci_repository_image_tag_filters = os.environ['soci_repository_image_tag_filters'].split(",")
    except:
        return log_and_generate_response(400, "Invalid environment variable; expected a comma separated list")

    for soci_repository_image_tag_filter in soci_repository_image_tag_filters:
        soci_repository_image_tag_filter_regex_pattern = fnmatch.translate(soci_repository_image_tag_filter)
        regex_matcher = re.compile(soci_repository_image_tag_filter_regex_pattern)

        if regex_matcher.fullmatch(f'{repository_name}:{image_tag}'):
            log_to_cloudwatch("Invoking SOCI index generator Lambda function")

            soci_index_generator_lambda_arn = os.environ['soci_index_generator_lambda_arn']

            response = lambda_client.invoke(
                FunctionName = soci_index_generator_lambda_arn,
                InvocationType = 'Event',
                Payload = json.dumps(event)
            )

            lambda_status_code = response['ResponseMetadata']['HTTPStatusCode']
            lambda_request_id = response['ResponseMetadata']['RequestId']

            if lambda_status_code == 202:
                return log_and_generate_response(200, f'Successfully invoked SOCI index generator Lambda function because the given event contained the image "{repository_name}:{image_tag}" with digest "{image_digest}" which matched SOCI repository image tag filter "{soci_repository_image_tag_filter}". Lambda request id: "{lambda_request_id}"')
            else:
                return log_and_generate_response(500, f'Failed to invoke SOCI index generator Lambda function. Lambda status code: "{lambda_status_code}". Lambda request id: "{lambda_request_id}"')

    return log_and_generate_response(200, f'The given event contained the image "{repository_name}:{image_tag}" with digest "{image_digest}" which did not match any SOCI repository image tag filters')

def log_and_generate_response(status_code, message):
    log_to_cloudwatch(message)

    return {
        'statusCode': status_code,
        'body': message
    }

def log_to_cloudwatch(message):
    print(message)