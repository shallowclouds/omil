package metric

type Client interface {
	Metric(name string, tags map[string]string, value map[string]interface{})
}
