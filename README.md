# AWS Serverless Tokenization Example

This project is an example of how to use serverless technologies to securely run
a PCI compliant tokenization service using AWS services that include Amazon API
Gateway, Amazon CloudWatch Logs, Amazon DynamoDB, AWS Key Management Service,
and AWS Lambda. This is meant to be a foundation for ideas; this project should
be forked and adjusted to an organization's needs before actually ever being
used for production.

## Security Philosophy

This project treats security as a constant moving target and uses rotating keys
managed through the AWS Key Management Service as well as a tokenization
algorithm that applies encryption on top of the hashed PAN that is salted
through the current policy at time of hash. The token encryption key is
periodically rotated which does a batch transactional update to the tokens in
the Amazon DynamoDB table with no customer impact. Also, if the token table gets
stolen the token encryption key can be rotated ad-hoc which also has no customer
impact. Furthermore when interacting with Amazon DynamoDB the data will
encrypted in transit as well as at rest.

## Policy Change

The policy is essentially the salt that is used to compute the hash of the
sensitive part of the PAN. This policy will not rotate because of the client
needing to maintain an immutable reference to the token but will instead evolve
by changing for new records at certain points in time.
