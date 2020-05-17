package quality_controller

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"

	"github.com/genomagic/config_parser"
	"github.com/genomagic/constants"
)

// adapterTrimming is the struct representation of the adapter trimming process
type adapterTrimming struct {
	// dockerCLI is used for launching a Docker container that perform adapter trimming
	dockerCLI *client.Client
	// config is the GenoMagic global configuration
	config *config_parser.Config
	// the context of the process
	ctx context.Context
}

func NewAdapterTrimming(ctx context.Context, dockerCli *client.Client, config *config_parser.Config) Controller {
	return &adapterTrimming{
		dockerCLI: dockerCli,
		config:    config,
		ctx:       ctx,
	}
}

// TODO: Replace this function into a constants file where this can be used by other quality control processes
// getImageID attempts to find the Docker image by given term
func getImageID(client *client.Client, ctx context.Context, term string) (string, error) {
	images, err := client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get available Docker images, err: %v", err)
	} else if len(images) == 0 {
		return "", fmt.Errorf("getImageID found no images")
	}
	found := false
	assemblerID := ""
	for _, im := range images {
		if found {
			break
		}
		for _, tag := range im.RepoTags {
			if strings.Contains(tag, term) {
				found = true
				assemblerID = im.ID
			}
		}
	}
	if !found {
		return "", fmt.Errorf("failed to find a Docker container for the given assembler")
	}
	return assemblerID, nil
}

func (a *adapterTrimming) Process() (string, error) {

	// TrimmedFileName is the filename where the trimmed reads are going to be stored.
	var TrimmedFileName = path.Join(constants.BaseOut, "trimmed.fastq")

	img, err := getImageID(a.dockerCLI, a.ctx, "replikation/porechop")
	if err != nil {
		return "", fmt.Errorf("cannot get image ID for fjukstad/trimmomatic, err: %v", err)
	}

	ctConfig := &container.Config{
		Tty: true,
		Cmd: []string{
			"-i", constants.RawSeqIn,
			"-o", TrimmedFileName,
		},
		Image: img,
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{ // Binding the input raw sequence file provided by the user
				Type:   mount.TypeBind,
				Source: a.config.GenoMagic.InputFilePath,
				Target: constants.RawSeqIn,
			},
			{ // Binding the output directory path provided by the user for saving trimmed file in.
				Type:   mount.TypeBind,
				Source: a.config.GenoMagic.OutputPath,
				Target: constants.BaseOut,
			},
		},
	}

	resp, err := a.dockerCLI.ContainerCreate(a.ctx, ctConfig, hostConfig, nil, "")
	if err != nil {
		return "", fmt.Errorf("fialed to create container, err: %v", err)
	}

	if err := a.dockerCLI.ContainerStart(a.ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container, err: %v", err)
	}

	statCh, errCh := a.dockerCLI.ContainerWait(a.ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("failed to wait for container to start up, err: %v", err)
		}
	case <-statCh:
	}

	out, err := a.dockerCLI.ContainerLogs(a.ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return "", fmt.Errorf("failed to get container log, err: %v", err)
	}

	if _, err := io.Copy(os.Stdout, out); err != nil {
		return "", fmt.Errorf("failed to capture stdout from Docker assembly container, err: %v", err)
	}
	return TrimmedFileName, nil
}
