// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package events

type ECRImageActionEventDetail struct {
	Result         string `json:"result"`
	RepositoryName string `json:"repository-name"`
	ImageDigest    string `json:"image-digest"`
	ActionType     string `json:"action-type"`
	ImageTag       string `json:"image-tag"`
}

type ECRImageActionEvent struct {
	Version    string                    `json:"version"`
	Id         string                    `json:"id"`
	DetailType string                    `json:"detail-type"`
	Source     string                    `json:"source"`
	Account    string                    `json:"account"`
	Time       string                    `json:"time"`
	Region     string                    `json:"region"`
	Resources  []string                  `json:"resources"`
	Detail     ECRImageActionEventDetail `json:"detail"`
}
