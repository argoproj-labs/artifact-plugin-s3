package s3

import (
	"context"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePluginConfiguration(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.Debug, logging.JSON))

	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		validate    func(t *testing.T, config *wfv1.S3Bucket)
	}{
		{
			name: "basic configuration",
			configYAML: `
bucket: my-bucket
endpoint: minio:9000
region: us-east-1
insecure: true
useSDKCreds: false
`,
			expectError: false,
			validate: func(t *testing.T, config *wfv1.S3Bucket) {
				assert.Equal(t, "my-bucket", config.Bucket)
				assert.Equal(t, "minio:9000", config.Endpoint)
				assert.Equal(t, "us-east-1", config.Region)
				assert.NotNil(t, config.Insecure)
				assert.True(t, *config.Insecure)
				assert.False(t, config.UseSDKCreds)
			},
		},
		{
			name: "configuration with secrets",
			configYAML: `
bucket: my-bucket
endpoint: minio:9000
insecure: true
accessKeySecret:
  name: my-minio-cred
  key: accesskey
secretKeySecret:
  name: my-minio-cred
  key: secretkey
`,
			expectError: false,
			validate: func(t *testing.T, config *wfv1.S3Bucket) {
				assert.Equal(t, "my-bucket", config.Bucket)
				assert.Equal(t, "minio:9000", config.Endpoint)

				// Check AccessKeySecret
				require.NotNil(t, config.AccessKeySecret)
				assert.Equal(t, "my-minio-cred", config.AccessKeySecret.Name)
				assert.Equal(t, "accesskey", config.AccessKeySecret.Key)

				// Check SecretKeySecret
				require.NotNil(t, config.SecretKeySecret)
				assert.Equal(t, "my-minio-cred", config.SecretKeySecret.Name)
				assert.Equal(t, "secretkey", config.SecretKeySecret.Key)
			},
		},
		{
			name: "configuration with session token",
			configYAML: `
bucket: my-bucket
endpoint: minio:9000
accessKeySecret:
  name: my-minio-cred
  key: accesskey
secretKeySecret:
  name: my-minio-cred
  key: secretkey
sessionTokenSecret:
  name: my-minio-cred
  key: sessiontoken
`,
			expectError: false,
			validate: func(t *testing.T, config *wfv1.S3Bucket) {
				assert.Equal(t, "my-bucket", config.Bucket)

				// Check all three secrets
				require.NotNil(t, config.AccessKeySecret)
				assert.Equal(t, "my-minio-cred", config.AccessKeySecret.Name)
				assert.Equal(t, "accesskey", config.AccessKeySecret.Key)

				require.NotNil(t, config.SecretKeySecret)
				assert.Equal(t, "my-minio-cred", config.SecretKeySecret.Name)
				assert.Equal(t, "secretkey", config.SecretKeySecret.Key)

				require.NotNil(t, config.SessionTokenSecret)
				assert.Equal(t, "my-minio-cred", config.SessionTokenSecret.Name)
				assert.Equal(t, "sessiontoken", config.SessionTokenSecret.Key)
			},
		},
		{
			name: "configuration with optional field",
			configYAML: `
bucket: my-bucket
endpoint: minio:9000
accessKeySecret:
  name: my-minio-cred
  key: accesskey
  optional: true
`,
			expectError: false,
			validate: func(t *testing.T, config *wfv1.S3Bucket) {
				require.NotNil(t, config.AccessKeySecret)
				assert.Equal(t, "my-minio-cred", config.AccessKeySecret.Name)
				assert.Equal(t, "accesskey", config.AccessKeySecret.Key)
				require.NotNil(t, config.AccessKeySecret.Optional)
				assert.True(t, *config.AccessKeySecret.Optional)
			},
		},
		{
			name: "configuration with unknown field (strict mode)",
			configYAML: `
bucket: my-bucket
endpoint: minio:9000
unknownField: value
`,
			expectError: true,
			validate:    nil,
		},
		{
			name: "minimal SDK credentials config",
			configYAML: `
bucket: my-bucket
endpoint: minio:9000
useSDKCreds: true
`,
			expectError: false,
			validate: func(t *testing.T, config *wfv1.S3Bucket) {
				assert.Equal(t, "my-bucket", config.Bucket)
				assert.Equal(t, "minio:9000", config.Endpoint)
				assert.True(t, config.UseSDKCreds)
				assert.Nil(t, config.AccessKeySecret)
				assert.Nil(t, config.SecretKeySecret)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := parsePluginConfiguration(ctx, tt.configYAML)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, config)
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}

// TestParsePluginConfiguration_EdgeCases tests edge cases and error conditions
func TestParsePluginConfiguration_EdgeCases(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.Debug, logging.JSON))

	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty YAML",
			configYAML:  "",
			expectError: false, // Empty YAML should create empty struct
		},
		{
			name:        "invalid YAML syntax",
			configYAML:  "bucket: my-bucket\n  invalid: [",
			expectError: true,
			errorMsg:    "failed to parse plugin configuration",
		},
		{
			name: "invalid secret structure",
			configYAML: `
bucket: my-bucket
accessKeySecret: "invalid-string-instead-of-object"
`,
			expectError: true,
			errorMsg:    "failed to parse plugin configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := parsePluginConfiguration(ctx, tt.configYAML)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
			}
		})
	}
}

// TestSecretKeySelector_FieldMapping verifies the YAML field mapping works correctly
func TestSecretKeySelector_FieldMapping(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.Debug, logging.JSON))

	// Test the exact YAML you're using in production
	configYAML := `bucket: my-bucket
endpoint: minio:9000
insecure: true
accessKeySecret:
  name: my-minio-cred
  key: accesskey
secretKeySecret:
  name: my-minio-cred
  key: secretkey`

	config, err := parsePluginConfiguration(ctx, configYAML)
	require.NoError(t, err)
	require.NotNil(t, config)

	t.Logf("Parsed config: %+v", config)

	// Debug the AccessKeySecret field specifically
	if config.AccessKeySecret != nil {
		t.Logf("AccessKeySecret: Name=%s, Key=%s, Optional=%v",
			config.AccessKeySecret.Name,
			config.AccessKeySecret.Key,
			config.AccessKeySecret.Optional)
	} else {
		t.Error("AccessKeySecret is nil")
	}

	// Debug the SecretKeySecret field specifically
	if config.SecretKeySecret != nil {
		t.Logf("SecretKeySecret: Name=%s, Key=%s, Optional=%v",
			config.SecretKeySecret.Name,
			config.SecretKeySecret.Key,
			config.SecretKeySecret.Optional)
	} else {
		t.Error("SecretKeySecret is nil")
	}
}
