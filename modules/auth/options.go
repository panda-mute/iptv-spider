package auth

type ClientOption func(*Client)

func WithUserAgent(ua string) ClientOption {
	return func(c *Client) {
		c.userAgent = ua
	}
}

func With4kLogAuthAddr(addr string) ClientOption {
	return func(c *Client) {
		c.pre4kLogAuthAddr = addr
	}
}

func WithSTBType(stbType string) ClientOption {
	return func(c *Client) {
		c.stbType = stbType
	}
}
