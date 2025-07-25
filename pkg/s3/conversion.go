package s3

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/pipekit/artifact-plugin-s3/pkg/artifact"
)

// ConvertToArgoArtifact converts protobuf artifact to Argo's artifact format
func ConvertToArgoArtifact(artifact *artifact.Artifact) *wfv1.Artifact {
	if artifact == nil {
		return nil
	}

	argoArtifact := &wfv1.Artifact{
		Name:     artifact.Name,
		Path:     artifact.Path,
		Optional: artifact.Optional,
		SubPath:  artifact.SubPath,
		Deleted:  artifact.Deleted,
	}

	// Convert artifact location if present
	if artifact.ArtifactLocation != nil {
		argoArtifact.ArtifactLocation = convertArtifactLocation(artifact.ArtifactLocation)
	}

	return argoArtifact
}

// convertArtifactLocation converts protobuf artifact location to Argo's format
func convertArtifactLocation(location *artifact.ArtifactLocation) wfv1.ArtifactLocation {
	argoLocation := wfv1.ArtifactLocation{
		ArchiveLogs: &location.ArchiveLogs,
	}

	// Convert S3 location if present
	if location.S3 != nil {
		argoLocation.S3 = convertS3Artifact(location.S3)
	}

	// Convert other artifact types as needed
	// For now, we only support S3

	return argoLocation
}

// convertS3Artifact converts protobuf S3 artifact to Argo's S3 artifact format
func convertS3Artifact(s3Artifact *artifact.S3Artifact) *wfv1.S3Artifact {
	if s3Artifact == nil {
		return nil
	}

	return &wfv1.S3Artifact{
		S3Bucket: wfv1.S3Bucket{
			Bucket: s3Artifact.Bucket,
		},
		Key: s3Artifact.Key,
	}
}
