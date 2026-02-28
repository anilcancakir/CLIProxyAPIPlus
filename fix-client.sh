#!/bin/bash
sed -i '' 's|_, err := c.patch("/v0/management/auth-files/fields", strings.NewReader(string(body)))|// TODO: endpoint not yet ported\n\treturn fmt.Errorf("Not implemented in origin")|g' internal/tui/client.go
