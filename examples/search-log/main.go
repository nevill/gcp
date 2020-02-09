package main

import (
	"fmt"
	"regexp"

	"github.com/nevill/gcp/logging"
)

const (
	project string = "projects/macro-mile-203600"
	image   string = "nevill/funcbench"
)

func main() {
	logManager, err := logging.New(project)
	if err != nil {
		panic(err)
	}

	condition := fmt.Sprintf("%s AND %s AND %s",
		"resource.type=gce_instance",
		fmt.Sprintf("logName=(%s/logs/cos_system)", project),
		fmt.Sprintf(`jsonPayload.MESSAGE:"%s"`, image),
	)

	// message is like:
	// 2020-02-08T15:47:20.855622008Z container die 90a71f7cdcd05a3d5657f147ca18dbd66e5ec8a2105288705cf16a28f6c0ee6b (exitCode=1, image=docker.io/nevill/funcbench:latest, name=klt-funcbench-test-fujb)
	err = logManager.Watch(condition, func(message string) bool {
		re := regexp.MustCompile(
			fmt.Sprintf(`container die (?P<id>\w{64}) \(exitCode=(?P<code>\d+), image=%s.+\)`, image),
		)
		result := re.FindStringSubmatch(message)
		if result != nil {
			fmt.Println(message)
		}
		return result != nil
	})

	if err != nil {
		panic(err)
	}
}
