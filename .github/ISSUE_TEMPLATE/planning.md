---
name: Planning
about: Planning
title: Planning
labels: ''
assignees: ''

---

The AWS-ia team has developed foundational asset to accelerate your development. You can use these assets to rapidly develop your architecture. These building block and tools will allow you to focus on your core product and import the underlying infrastructure.  By nesting these templates you can provide multiple entry points for customer and maintain just your workload. 

### Add submodules for (foundational assets)

Foundational assets are Quick Start building Blocks. Partners can import these assets into their design as a git submodule. (Offload the maintenance of this code to the upstream repo which is open sourced under Apache 2.0)
 
Examples of foundational assets
* VPC
* EKS
* RDS
* Bastion
* (etc)


See full list of foundation Quick Starts https://aws.amazon.com/quickstart/

### Submodule usage

#### Example 1 ( EC2 based deployment)  
```/templates
├── main-yaml.template #VPC Entry point
└── workload-yaml.template #Workload Entry point
```
Provides 2 Entry point for ec2 based deployment end user can choose to use existing VPC or Create new VPC and deploy into it
**VPC Entry Point** : Create VPC and passes VPC info to Workload template 
**Workload template**: Expects VPC information to be passed in and install partner products into the VPC

#### Example 2 (Deploy using EKS)
```/templates
├── main-yaml.template #VPC Entry point (Option to create EKS Cluster via conditional)
└── workload-yaml.template #Workload Entry point
```
Similarly you can submodule the EKS Quick Start and provide additional Entry points
**Create new EKS cluster in existing VPC:** Create EKS Cluster and deploy partner product using helm
**Use existing VPC and EKS Cluster: ** Workload templates expects EKS cluster and VPC info

### How to I add a submodule ?

This example references two submodule to repo (AWS Quick Start VPC and Linux Bastion)                       **Note: you must fork the repo into you own account and do the following step in you fork**

- [x] Fork this repo
- [x] Clone your fork 
- [x] Change to the root of your fork 

**To add the VPC as a submodule:**  
`git submodule add -b main https://github.com/aws-quickstart/quickstart-aws-vpc.git submodules/quickstart-aws-vpc`

**Update your fork**
`git submodule update --remote --merge && git submodule update —recursive`

**Commit your changes**
 `git commit -a -m 'Add VPC and Linux Bastion submodules'`

**Referencing you submodule in the main template.** 

```Resources:
  VPCStack:
    Type: 'AWS::CloudFormation::Stack'
    Properties:
      TemplateURL: !Sub
        - https://${S3Bucket}.s3.${S3Region}.${AWS::URLSuffix}/${QSS3KeyPrefix}submodules/quickstart-aws-vpc/templates/aws-vpc.template.yaml
        - S3Bucket: !If
            - UsingDefaultBucket
            - !Sub 'aws-quickstart-${AWS::Region}'
            - !Ref 'QSS3BucketName'
          S3Region: !If
            - UsingDefaultBucket
            - !Ref 'AWS::Region'
            - !Ref 'QSS3BucketRegion'
      Parameters:
        AvailabilityZones: !Join 
          - ','
          - !Ref AvailabilityZones
        NumberOfAZs: '2'
        PrivateSubnet1ACIDR: !Ref PrivateSubnet1CIDR
        PrivateSubnet2ACIDR: !Ref PrivateSubnet2CIDR
        PublicSubnet1CIDR: !Ref PublicSubnet1CIDR
        PublicSubnet2CIDR: !Ref PublicSubnet2CIDR
        VPCCIDR: !Ref VPCCIDR
        VPCTenancy: !Ref VPCTenancy
```

> Functional Example here https://github.com/aws-quickstart/quickstart-linux-bastion/blob/main/templates/linux-bastion-master.template#L260-L284

> IF your design does not require any underlying infrastructure (i.e Serverless or SaaS based) you can highlight that point in the issue comments and resolve this issue
