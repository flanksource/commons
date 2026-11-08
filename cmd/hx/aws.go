package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/flanksource/commons/http/middlewares/awssigv4"
)

func sdkCredentialsProvider(provider aws.CredentialsProvider) awssigv4.CredentialsProvider {
	return awssigv4.CredentialsProviderFunc(func(ctx context.Context) (awssigv4.Credentials, error) {
		value, err := provider.Retrieve(ctx)
		if err != nil {
			return awssigv4.Credentials{}, err
		}
		result := awssigv4.Credentials{AccessKeyID: value.AccessKeyID, SecretAccessKey: value.SecretAccessKey, SessionToken: value.SessionToken}
		if value.CanExpire {
			result.Expires = value.Expires
		}
		return result, nil
	})
}
