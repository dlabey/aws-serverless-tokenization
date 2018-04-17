# AWS Serverless Tokenization Example

This project is an example of how to use serverless technologies to securely run
a PCI compliant tokenization service using AWS services that include Amazon API
Gateway, Amazon CloudWatch Logs, Amazon DynamoDB, AWS Key Management Service,
and AWS Lambda. This is meant to be a foundation for ideas; this project should
be forked and adjusted to an organization's needs before actually ever being
used for production.

## Security Philosophy

This project treats security as a constant moving target and uses evolving
policies and rotating keys. Policies are used in calculating the token and keys
are used in encrypting the PAN. The token is calculated in a way that allows
determination of the BIN and SPAN while maintaining confidentiality at the table
level through use of the policy. The policy is periodically evolved to provide
an extra layer of segmentation and obfuscation for the token. The PAN encryption
keys are managed through the AWS Key Management Service for encryption of the
PAN. The keys are periodically rotated which does a batch transactional update
to the PANs in the Amazon DynamoDB table with no customer impact. Also, if the
token table gets stolen the token encryption key can be rotated ad-hoc which
also has no customer impact. Furthermore, when interacting with Amazon DynamoDB
the data is encrypted in transit as well as at rest.

## Policy Change

The policy is essentially the key that is used to encrypt the BIN and SPAN as
well as the salt that is used compute a unique hash of the PAN in the token.
The policy does not rotate because of the client needing to maintain an
immutable reference to the token but instead evolves by changing for new records
at certain points in time.
