package unitski

import (
	"bufio"
	"context"
	"github.com/docker/docker/client"
	"log"
	"os"
	"os/exec"
	"strings"
)

type DockerError struct {
	msg string
}

func (error *DockerError) Error() string {
	return error.msg
}

func InitDocker() (*client.Client, context.Context) {
	// Create the docker client
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	return cli, ctx
}

// DumpMySqlDatabase dumps the database from a docker container that is running MySQL/MariaDB
func DumpMySqlDatabase(cli *client.Client, ctx context.Context, config BackupConfigDatabase, dumpToFile string) (err error) {
	// Get all information about the container
	containerId := config.Container
	container, err := cli.ContainerInspect(ctx, containerId)
	if err != nil {
		return err
	}

	// Check if the container is running
	if !container.State.Running {
		return &DockerError{"Container " + containerId + " (db: " + config.Name + ") isn't running!"}
	}

	// Determine the required mysqldump variables
	env := parseEnvVariables(container.Config.Env)
	database, err := getEnvOrDefault(env, "", config.Database)
	if err != nil {
		return err
	}
	user, err := getEnvOrDefault(env, "root", config.User)
	if err != nil {
		return err
	}
	password, err := getEnvOrDefault(env, "", config.Password)
	if err != nil {
		return err
	}

	// Create the file to write to
	outfile, err := os.Create(dumpToFile)
	if err != nil {
		return err
	}
	defer outfile.Close()

	// Attempt to dump the database
	dump := exec.Command(
		"docker",
		"exec",
		containerId,
		"mysqldump",
		"-u",
		user,
		"-p"+password+"",
		database,
	)

	// Output should be to the file
	dump.Stdout = outfile

	// Capture any error output
	stderr, err := dump.StderrPipe()
	if err != nil {
		return err
	}

	// Run the command
	err = dump.Start()
	if err != nil {
		_ = os.Remove(dumpToFile)
		return err
	}
	defer func() {
		if err = dump.Wait(); err != nil {
			_ = os.Remove(dumpToFile)
		}
	}()

	// Read any possible error lines
	errBuffer := bufio.NewScanner(stderr)
	for errBuffer.Scan() {
		log.Println(errBuffer.Text())
	}

	return err
}

func parseEnvVariables(env []string) map[string]string {
	result := map[string]string{}
	for _, envValue := range env {
		split := strings.SplitN(envValue, "=", 2)
		result[split[0]] = split[1]
	}
	return result
}

func getEnvOrDefault(env map[string]string, defaultValue string, variable BackupVariable) (string, error) {
	// Check if the variable was set
	if variable.VarType == "" || variable.Value == "" {
		return defaultValue, nil
	}

	// Might a hard / constant value
	if variable.VarType == VarTypeConstant {
		return variable.Value, nil
	}

	// Otherwise, it should be fetched from the env
	if envValue, ok := env[variable.Value]; ok {
		return envValue, nil
	} else {
		return "", &DockerError{"Was unable to find docker env '" + variable.Value + "' on container"}
	}
}
