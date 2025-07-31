package s3

import (
	"context"
	"fmt"
	"os"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"
)

// parsePluginConfiguration parses YAML configuration from Plugin.Configuration string
func parsePluginConfiguration(ctx context.Context, configYAML string) (*wfv1.S3Bucket, error) {
	var config wfv1.S3Bucket

	// Use Kubernetes SIGS YAML which is more compatible with Kubernetes API types
	err := yaml.UnmarshalStrict([]byte(configYAML), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plugin configuration: %w", err)
	}

	logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{
		"input":  configYAML,
		"output": config,
	}).Debug(ctx, "Parsed plugin configuration")

	return &config, nil
}

func DriverAndArtifactFromConfig(ctx context.Context, configYaml string, key string) (*ArtifactDriver, *wfv1.Artifact, error) {
	pluginConfig, err := parsePluginConfiguration(ctx, configYaml)
	if err != nil {
		return nil, nil, err
	}

	artifact := createArgoArtifactFromConfig(pluginConfig, key)
	driver, err := getArtifactDriver(ctx, pluginConfig)

	return driver, artifact, err
}

func createArgoArtifactFromConfig(pluginConfig *wfv1.S3Bucket, key string) *wfv1.Artifact {
	return &wfv1.Artifact{
		ArtifactLocation: wfv1.ArtifactLocation{
			S3: &wfv1.S3Artifact{
				S3Bucket: *pluginConfig,
				Key:      key,
			},
		},
	}
}

func getArtifactDriver(ctx context.Context, pluginConfig *wfv1.S3Bucket) (*ArtifactDriver, error) {
	// Create base ArtifactDriver from plugin config
	driver := &ArtifactDriver{
		Endpoint:    pluginConfig.Endpoint,
		Region:      pluginConfig.Region,
		Secure:      pluginConfig.Insecure == nil || !*pluginConfig.Insecure, // Insecure is inverted to Secure
		RoleARN:     pluginConfig.RoleARN,
		UseSDKCreds: pluginConfig.UseSDKCreds,
	}

	// If UseSDKCreds is true, we don't need to resolve any secrets
	if pluginConfig.UseSDKCreds {
		return driver, nil
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
	if pluginConfig.AccessKeySecret != nil {
		accessKey, err := getSecretValue(ctx, clientset, pluginConfig.AccessKeySecret.Name, pluginConfig.AccessKeySecret.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve access key: %w", err)
		}
		driver.AccessKey = accessKey
	}

	// Resolve secret key
	if pluginConfig.SecretKeySecret != nil {
		secretKey, err := getSecretValue(ctx, clientset, pluginConfig.SecretKeySecret.Name, pluginConfig.SecretKeySecret.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve secret key: %w", err)
		}
		driver.SecretKey = secretKey
	}

	// Resolve session token (optional)
	if pluginConfig.SessionTokenSecret != nil {
		sessionToken, err := getSecretValue(ctx, clientset, pluginConfig.SessionTokenSecret.Name, pluginConfig.SessionTokenSecret.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve session token: %w", err)
		}
		driver.SessionToken = sessionToken
	}

	logging.RequireLoggerFromContext(ctx).WithField("driver", driver).Debug(ctx, "Resolved S3 configuration")

	return driver, nil
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
