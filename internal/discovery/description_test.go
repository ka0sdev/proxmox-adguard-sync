package discovery

import "testing"

func TestParseDescription(t *testing.T) {
	description := `
Application proxy

dns_ip=172.20.0.8
dns_name=lxc-proxy-01
`

	result := ParseDescription(
		description,
		[]string{"dns_ip", "ip"},
		[]string{"dns_name", "name"},
	)

	if !result.HasAddress {
		t.Fatal("HasAddress = false, expected true")
	}

	if result.Address.String() != "172.20.0.8" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.0.8",
		)
	}

	if result.AddressKey != "dns_ip" {
		t.Errorf(
			"AddressKey = %q, expected %q",
			result.AddressKey,
			"dns_ip",
		)
	}

	if !result.HasName {
		t.Fatal("HasName = false, expected true")
	}

	if result.Name != "lxc-proxy-01" {
		t.Errorf(
			"Name = %q, expected %q",
			result.Name,
			"lxc-proxy-01",
		)
	}

	if result.NameKey != "dns_name" {
		t.Errorf(
			"NameKey = %q, expected %q",
			result.NameKey,
			"dns_name",
		)
	}
}

func TestParseDescriptionUsesConfiguredPrecedence(
	t *testing.T,
) {
	description := `
ip=172.20.0.20
dns_ip=172.20.0.10
name=fallback-name
dns_name=preferred-name
`

	result := ParseDescription(
		description,
		[]string{"dns_ip", "ip"},
		[]string{"dns_name", "name"},
	)

	if result.Address.String() != "172.20.0.10" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.0.10",
		)
	}

	if result.Name != "preferred-name" {
		t.Errorf(
			"Name = %q, expected %q",
			result.Name,
			"preferred-name",
		)
	}
}

func TestParseDescriptionSupportsCaseInsensitiveKeys(
	t *testing.T,
) {
	description := `
DNS_IP=172.20.0.9/16
DNS_NAME=Database
`

	result := ParseDescription(
		description,
		[]string{"dns_ip"},
		[]string{"dns_name"},
	)

	if !result.HasAddress {
		t.Fatal("HasAddress = false, expected true")
	}

	if result.Address.String() != "172.20.0.9" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.0.9",
		)
	}

	if result.Name != "Database" {
		t.Errorf(
			"Name = %q, expected %q",
			result.Name,
			"Database",
		)
	}
}

func TestParseDescriptionSupportsWindowsLineEndings(
	t *testing.T,
) {
	description := "dns_ip=172.20.0.7\r\ndns_name=edge\r\n"

	result := ParseDescription(
		description,
		[]string{"dns_ip"},
		[]string{"dns_name"},
	)

	if result.Address.String() != "172.20.0.7" {
		t.Errorf(
			"Address = %q, expected %q",
			result.Address,
			"172.20.0.7",
		)
	}

	if result.Name != "edge" {
		t.Errorf(
			"Name = %q, expected %q",
			result.Name,
			"edge",
		)
	}
}

func TestParseDescriptionIgnoresInvalidAddress(
	t *testing.T,
) {
	result := ParseDescription(
		"dns_ip=invalid\ndns_name=test",
		[]string{"dns_ip"},
		[]string{"dns_name"},
	)

	if result.HasAddress {
		t.Fatal("HasAddress = true, expected false")
	}

	if !result.HasName {
		t.Fatal("HasName = false, expected true")
	}
}

func TestParseDescriptionIgnoresIPv6Address(
	t *testing.T,
) {
	result := ParseDescription(
		"dns_ip=2001:db8::1",
		[]string{"dns_ip"},
		nil,
	)

	if result.HasAddress {
		t.Fatal("HasAddress = true, expected false")
	}
}

func TestParseDescriptionAllowsEqualsInValue(
	t *testing.T,
) {
	result := ParseDescription(
		"dns_name=application=primary",
		nil,
		[]string{"dns_name"},
	)

	if result.Name != "application=primary" {
		t.Errorf(
			"Name = %q, expected %q",
			result.Name,
			"application=primary",
		)
	}
}

func TestParseDescriptionReturnsEmptyResult(
	t *testing.T,
) {
	result := ParseDescription(
		"ordinary description text",
		[]string{"dns_ip", "ip"},
		[]string{"dns_name", "name"},
	)

	if result.HasAddress {
		t.Fatal("HasAddress = true, expected false")
	}

	if result.HasName {
		t.Fatal("HasName = true, expected false")
	}
}
