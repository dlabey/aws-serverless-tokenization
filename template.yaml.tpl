AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31
Description: AWS Serverless Tokenization Example

Resources:
  # DynamoDB Tables
  EncryptionKeys:
    Type: AWS::DynamoDB::Table
    Properties:
      AttributeDefinitions:
        - AttributeName: EncryptionKeyID
          AttributeType: S
        # - AttributeName: DataKey
        #   AttributeType: Binary
        # - AttributeName: IsCurrent
        #   AttributeType: BOOL
        # - AttributeName: RotatedAt
        #   AttributeType: S
      KeySchema:
        - AttributeName: EncryptionKeyID
          KeyType: HASH
      ProvisionedThroughput:
        ReadCapacityUnits: 5
        WriteCapacityUnits: 5
      TableName: EncryptionKeys
      SSESpecification:
        SSEEnabled: true

  PolicyKeys:
    Type: AWS::DynamoDB::Table
    Properties:
      AttributeDefinitions:
        - AttributeName: PolicyKeyID
          AttributeType: S
        # - AttributeName: DataKey
        #   AttributeType: Binary
        # - AttributeName: CreatedAt
        #   AttributeType: S
      KeySchema:
        - AttributeName: PolicyKeyID
          KeyType: HASH
      ProvisionedThroughput:
        ReadCapacityUnits: 5
        WriteCapacityUnits: 5
      TableName: PolicyKeys
      SSESpecification:
        SSEEnabled: true

  Tokens:
    Type: AWS::DynamoDB::Table
    Properties:
      AttributeDefinitions:
        - AttributeName: Token
          AttributeType: S
        # - AttributeName: PolicyKeyId
        #   AttributeType: S
        # - AttributeName: EncryptionKeyId
        #   AttributeType: S
        # - AttributeName: Pan
        #   AttributeType: S
      KeySchema:
        - AttributeName: Token
          KeyType: HASH
      ProvisionedThroughput:
        ReadCapacityUnits: 5
        WriteCapacityUnits: 5
      TableName: Tokens
      SSESpecification:
        SSEEnabled: true

  # Lambda Functions
  Tokenizer:
    Type: AWS::Serverless::Function
    Properties:
      Handler: cmd/tokenize/main
      Runtime: go1.x
      Events:
        Tokenize:
          Type: Api
          Properties:
            Path: /tokenize
            Method: post

  Detokenizer:
    Type: AWS::Serverless::Function
    Properties:
      Handler: cmd/detokenize/main
      Runtime: go1.x
      Events:
        Detokenize:
          Type: Api
          Properties:
            Path: /detokenize
            Method: post

  EncryptionKeyRotatorDocument:
    Type: AWS::SSM::Document
    Properties:
      DocumentType: Automation
      Content: !Sub |
        {
          "schemaVersion": "0.3",
          "parameters": {
              "IsReplace": {
                  "type": "String",
                  "default": "false"
              },
              "EncryptionKeyID": {
                  "type": "String",
                  "default": ""
              }
          },
          "mainSteps": [{
              "name": "InvokeEncryptionKeyRotator",
              "action": "aws:invokeLambdaFunction",
              "inputs": {
                  "FunctionName": "${EncryptionKeyRotator}",
                  "Payload": "{\"IsReplace\": \"{{IsReplace}}\", \"EncryptionKeyID\": \"{{EncryptionKeyID}}\"}"
              }
          }]
        }
      
  EncryptionKeyRotator:
    Type: AWS::Serverless::Function
    Properties:
      Handler: cmd/rotate_encryption_key/main
      Runtime: go1.x
      Events:
        Rotate:
          Type: CloudWatchEvent
          Properties:
            Pattern:
              Source:
                - aws.ssm
              DetailType:
                - EC2 Command Status-change Notification
              Detail:
                DocumentName:
                  - EncryptionKeyRotatorDocument

  PolicyKeyEvolverDocument:
    Type: AWS::SSM::Document
    Properties:
      DocumentType: Automation
      Content: !Sub |
        {
          "schemaVersion": "0.3",
          "mainSteps": [{
              "name": "InvokePolicyKeyEvolver",
              "action": "aws:invokeLambdaFunction",
              "inputs": {
                  "FunctionName": "${PolicyKeyEvolver}"
              }
          }]
        }

  PolicyKeyEvolver:
    Type: AWS::Serverless::Function
    Properties:
      Handler: cmd/evolve_policy_key/main
      Runtime: go1.x
      Events:
        Evolve:
          Type: CloudWatchEvent
          Properties:
            Pattern:
              Source:
                - aws.ssm
              DetailType:
                - EC2 Command Status-change Notification
              Detail:
                DocumentName:
                  - PolicyKeyEvolverDocument