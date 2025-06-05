package client

const (
	attrWeight      = "attr_weight"
	attrReadWeight  = "attr_read_weight"
	attrWriteWeight = "attr_write_weight"
	attrGroup       = "attr_group"
	attrNode        = "attr_node"
)

type groupContextKey struct{}

type ContextKey string

const KeyRequestType ContextKey = "request_type"
