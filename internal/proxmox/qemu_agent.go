package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type QEMUAgentNetworkResult struct {
	Result []QEMUAgentInterface `json:"result"`
}

type QEMUAgentInterface struct {
	Name            string               `json:"name"`
	HardwareAddress string               `json:"hardware-address"`
	IPAddresses     []QEMUAgentIPAddress `json:"ip-addresses"`
}

type QEMUAgentIPAddress struct {
	Type    string `json:"ip-address-type"`
	Address string `json:"ip-address"`
	Prefix  int    `json:"prefix"`
}

func (c *Client) GetQEMUAgentInterfaces(
	ctx context.Context,
	node string,
	vmid int,
) ([]QEMUAgentInterface, error) {
	node = strings.TrimSpace(node)

	if node == "" {
		return nil, errors.New(
			"QEMU node must not be empty",
		)
	}

	if vmid <= 0 {
		return nil, errors.New(
			"QEMU VMID must be greater than zero",
		)
	}

	path := fmt.Sprintf(
		"/nodes/%s/qemu/%d/agent/network-get-interfaces",
		url.PathEscape(node),
		vmid,
	)

	var result QEMUAgentNetworkResult

	if err := c.get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf(
			"get QEMU Guest Agent interfaces for node %s VMID %d: %w",
			node,
			vmid,
			err,
		)
	}

	return result.Result, nil
}
