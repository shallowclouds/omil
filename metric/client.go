package metric

import "time"

type Client interface {
	Metric(name string, timestamp time.Time, tags map[string]string, value map[string]interface{})
	Flush() error
	Exit() error
}
