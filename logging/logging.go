package logging

import (
	"context"
	"fmt"
	"time"

	logging "cloud.google.com/go/logging/apiv2"
	"google.golang.org/api/iterator"
	loggingpb "google.golang.org/genproto/googleapis/logging/v2"
)

// Manager resprents the managed logging resources.
type Manager struct {
	project       string
	loggingClient *logging.Client
}

// New creates a new Manager.
func New(project string) (*Manager, error) {
	client, err := logging.NewClient(context.Background())
	if err != nil {
		return nil, err
	}

	manager := Manager{
		loggingClient: client,
		project:       project,
	}
	return &manager, nil
}

// MatchFunc tests if the message contains specified text.
// If yes, it will return true, otherwise return false.
type MatchFunc func(message string) bool

// Watch iterates over the container logs matches the condition, until function testEnd returns true.
func (m *Manager) Watch(condition string, testEnd MatchFunc) error {
	createRequest := func(beginAt time.Time) *loggingpb.ListLogEntriesRequest {
		filter := fmt.Sprintf("%s AND %s",
			condition,
			fmt.Sprintf(`timestamp > "%s"`, beginAt.Format(time.RFC3339)),
		)
		return &loggingpb.ListLogEntriesRequest{
			ResourceNames: []string{
				m.project,
			},
			Filter:  filter,
			OrderBy: "timestamp asc",
		}
	}

	req := createRequest(time.Now())

	//TODO improve context usage
	it := m.loggingClient.ListLogEntries(context.Background(), req)
	for ended := false; ended == false; {
		resp, err := it.Next()
		if err == iterator.Done {
			if ended == true {
				break
			}
			time.Sleep(1 * time.Minute)
			it = m.loggingClient.ListLogEntries(context.Background(), req)
			continue
		} else if err != nil {
			return err
		}
		message := resp.GetJsonPayload().Fields["MESSAGE"].GetStringValue()
		ended = testEnd(message)
	}
	return nil
}
