package aws

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"text/template"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	"github.com/muraduiurie/aws-ai-agent/pkg/types"
)

var kubeconfigTemplate = template.Must(template.New("kubeconfig").Parse(`apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: {{ .CAData }}
    server: {{ .Endpoint }}
  name: {{ .ClusterName }}
contexts:
- context:
    cluster: {{ .ClusterName }}
    user: {{ .ClusterName }}
  name: {{ .ClusterName }}
current-context: {{ .ClusterName }}
kind: Config
preferences: {}
users:
- name: {{ .ClusterName }}
  user:
    token: {{ .Token }}
`))

// ListEKSClusters returns the names of all EKS clusters in the given region.
func (f *Factory) ListEKSClusters(ctx context.Context, region string) ([]string, error) {
	client := eks.NewFromConfig(f.cfg, func(o *eks.Options) {
		o.Region = region
	})

	var names []string
	paginator := eks.NewListClustersPaginator(client, &eks.ListClustersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list EKS clusters: %w", err)
		}
		names = append(names, page.Clusters...)
	}

	return names, nil
}

// GetEKSCluster returns the details of a single EKS cluster.
func (f *Factory) GetEKSCluster(ctx context.Context, region, name string) (*types.EKSCluster, error) {
	client := eks.NewFromConfig(f.cfg, func(o *eks.Options) {
		o.Region = region
	})

	out, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: awssdk.String(name),
	})
	if err != nil {
		return nil, fmt.Errorf("describe EKS cluster %q: %w", name, err)
	}

	c := out.Cluster
	cluster := &types.EKSCluster{
		Name:     awssdk.ToString(c.Name),
		ARN:      awssdk.ToString(c.Arn),
		Status:   string(c.Status),
		Version:  awssdk.ToString(c.Version),
		Endpoint: awssdk.ToString(c.Endpoint),
		RoleARN:  awssdk.ToString(c.RoleArn),
	}
	if c.CreatedAt != nil {
		cluster.CreatedAt = c.CreatedAt.UTC().Format(time.RFC3339)
	}

	return cluster, nil
}

// GetEKSKubeconfig generates a kubeconfig string for the given EKS cluster.
// Authentication uses a short-lived token obtained via a presigned STS
// GetCallerIdentity request — the same mechanism as `aws eks get-token`.
func (f *Factory) GetEKSKubeconfig(ctx context.Context, region, name string) (string, error) {
	cluster, err := f.GetEKSCluster(ctx, region, name)
	if err != nil {
		return "", err
	}

	eksClient := eks.NewFromConfig(f.cfg, func(o *eks.Options) {
		o.Region = region
	})
	out, err := eksClient.DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: awssdk.String(name),
	})
	if err != nil {
		return "", fmt.Errorf("describe EKS cluster for kubeconfig: %w", err)
	}
	caData := awssdk.ToString(out.Cluster.CertificateAuthority.Data)

	token, err := f.generateEKSToken(ctx, region, name)
	if err != nil {
		return "", fmt.Errorf("generate EKS token: %w", err)
	}

	var buf bytes.Buffer
	if err := kubeconfigTemplate.Execute(&buf, map[string]string{
		"ClusterName": name,
		"Endpoint":    cluster.Endpoint,
		"CAData":      caData,
		"Token":       token,
	}); err != nil {
		return "", fmt.Errorf("render kubeconfig: %w", err)
	}

	return buf.String(), nil
}

// generateEKSToken produces a short-lived bearer token accepted by EKS.
// It presigns an STS GetCallerIdentity request with the cluster name header,
// then base64url-encodes the URL with the "k8s-aws-v1." prefix.
func (f *Factory) generateEKSToken(ctx context.Context, region, clusterName string) (string, error) {
	stsClient := sts.NewFromConfig(f.cfg, func(o *sts.Options) {
		o.Region = region
	})
	presign := sts.NewPresignClient(stsClient)

	req, err := presign.PresignGetCallerIdentity(ctx, &sts.GetCallerIdentityInput{},
		sts.WithPresignClientFromClientOptions(func(o *sts.Options) {
			o.APIOptions = append(o.APIOptions, func(stack *middleware.Stack) error {
				return stack.Build.Add(middleware.BuildMiddlewareFunc(
					"AddClusterNameHeader",
					func(ctx context.Context, in middleware.BuildInput, next middleware.BuildHandler) (middleware.BuildOutput, middleware.Metadata, error) {
						if r, ok := in.Request.(*smithyhttp.Request); ok {
							r.Header.Set("x-k8s-aws-id", clusterName)
						}
						return next.HandleBuild(ctx, in)
					},
				), middleware.Before)
			})
		}),
	)
	if err != nil {
		return "", fmt.Errorf("presign GetCallerIdentity: %w", err)
	}

	return "k8s-aws-v1." + base64.RawURLEncoding.EncodeToString([]byte(req.URL)), nil
}
