package s3

import (
	"context"
	"fmt"
	"os"

	"github.com/pipekit/artifact-plugin-s3/pkg/artifact"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// S3Config holds resolved S3 configuration with credentials
type S3Config struct {
	Endpoint     string
	Region       string
	Secure       bool
	AccessKey    string
	SecretKey    string
	SessionToken string
	RoleARN      string
	UseSDKCreds  bool
	Bucket       string
	Key          string
}

// extractS3Config converts protobuf S3Artifact to S3Config
func extractS3Config(s3Artifact *artifact.S3Artifact) *S3Config {
	if s3Artifact == nil {
		return nil
	}

	return &S3Config{
		Endpoint:    s3Artifact.Endpoint,
		Region:      s3Artifact.Region,
		Secure:      !s3Artifact.Insecure,
		RoleARN:     s3Artifact.RoleArn,
		UseSDKCreds: s3Artifact.UseSdkCreds,
		Bucket:      s3Artifact.Bucket,
		Key:         s3Artifact.Key,
	}
}

// ResolveCredentials resolves credentials from Kubernetes secrets
func ResolveCredentials(ctx context.Context, s3Artifact *artifact.S3Artifact) (*S3Config, error) {
	config := extractS3Config(s3Artifact)
	if config == nil {
		return nil, fmt.Errorf("invalid S3 configuration")
	}

	// If UseSDKCreds is true, we don't need to resolve any secrets
	if config.UseSDKCreds {
		return config, nil
	}

	// Create Kubernetes client
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Resolve access key
	if s3Artifact.AccessKeySecretName != "" && s3Artifact.AccessKeySecretKey != "" {
		accessKey, err := getSecretValue(ctx, clientset, s3Artifact.AccessKeySecretName, s3Artifact.AccessKeySecretKey)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve access key: %w", err)
		}
		config.AccessKey = accessKey
	}

	// Resolve secret key
	if s3Artifact.SecretKeySecretName != "" && s3Artifact.SecretKeySecretKey != "" {
		secretKey, err := getSecretValue(ctx, clientset, s3Artifact.SecretKeySecretName, s3Artifact.SecretKeySecretKey)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve secret key: %w", err)
		}
		config.SecretKey = secretKey
	}

	// Resolve session token (optional)
	if s3Artifact.SessionTokenSecretName != "" && s3Artifact.SessionTokenSecretKey != "" {
		sessionToken, err := getSecretValue(ctx, clientset, s3Artifact.SessionTokenSecretName, s3Artifact.SessionTokenSecretKey)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve session token: %w", err)
		}
		config.SessionToken = sessionToken
	}

	return config, nil
}

// getSecretValue retrieves a value from a Kubernetes secret
func getSecretValue(ctx context.Context, clientset *kubernetes.Clientset, secretName, secretKey string) (string, error) {
	// Get namespace from service account token
	namespace, err := getNamespace()
	if err != nil {
		return "", fmt.Errorf("failed to get namespace: %w", err)
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", secretName, err)
	}

	value, exists := secret.Data[secretKey]
	if !exists {
		return "", fmt.Errorf("secret key %s not found in secret %s", secretKey, secretName)
	}

	return string(value), nil
}

// getNamespace reads the namespace from the service account token
func getNamespace() (string, error) {
	// Read namespace from the mounted service account token
	namespaceBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", fmt.Errorf("failed to read namespace: %w", err)
	}
	return string(namespaceBytes), nil
}

// CreateArtifactDriver creates an ArtifactDriver from resolved S3 configuration
func CreateArtifactDriver(config *S3Config) *ArtifactDriver {
	return &ArtifactDriver{
		Endpoint:     config.Endpoint,
		Region:       config.Region,
		Secure:       config.Secure,
		AccessKey:    config.AccessKey,
		SecretKey:    config.SecretKey,
		SessionToken: config.SessionToken,
		RoleARN:      config.RoleARN,
		UseSDKCreds:  config.UseSDKCreds,
		Context:      context.Background(),
	}
}
