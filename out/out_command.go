package out

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/concourse/s3-resource"
	"github.com/concourse/s3-resource/versions"
)

type OutCommand struct {
	s3client s3resource.S3Client
}

func NewOutCommand(s3client s3resource.S3Client) *OutCommand {
	return &OutCommand{
		s3client: s3client,
	}
}

func (command *OutCommand) Run(sourceDir string, request OutRequest) (OutResponse, error) {
	match, err := command.match(sourceDir, request.Params.From)
	if err != nil {
		return OutResponse{}, err
	}

	var remotePath string

	folderDestination := strings.HasSuffix(request.Params.To, "/")
	if folderDestination || request.Params.To == "" {
		remotePath = filepath.Join(request.Params.To, filepath.Base(match))
	} else {
		compiled := regexp.MustCompile(request.Params.From)
		fileName := strings.TrimPrefix(match, sourceDir+"/")
		remotePath = compiled.ReplaceAllString(fileName, request.Params.To)
	}

	bucketName := request.Source.Bucket

	err = command.s3client.UploadFile(
		bucketName,
		remotePath,
		match,
	)
	if err != nil {
		return OutResponse{}, err
	}

	return OutResponse{
		Version: s3resource.Version{
			Path: remotePath,
		},
		Metadata: command.metadata(bucketName, remotePath, request.Source.Private),
	}, nil
}

func (command *OutCommand) match(sourceDir, pattern string) (string, error) {
	paths := []string{}
	filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		paths = append(paths, path)
		return nil
	})

	matches, err := versions.MatchUnanchored(paths, pattern)
	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no matches found for pattern: %s", pattern)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("more than one match found for pattern: %s\n%v", pattern, matches)
	}

	return matches[0], nil
}

func (command *OutCommand) metadata(bucketName, remotePath string, private bool) []s3resource.MetadataPair {
	remoteFilename := filepath.Base(remotePath)

	metadata := []s3resource.MetadataPair{
		s3resource.MetadataPair{
			Name:  "filename",
			Value: remoteFilename,
		},
	}

	if !private {
		metadata = append(metadata, s3resource.MetadataPair{
			Name:  "url",
			Value: command.s3client.URL(bucketName, remotePath, false),
		})
	}

	return metadata
}
