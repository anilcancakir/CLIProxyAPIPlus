#!/bin/bash
find internal/tui -type f -name "*.go" -exec sed -i '' 's|"github.com/kooshapari/cliproxyapi-plusplus/v6/pkg/llmproxy/misc"|"github.com/router-for-me/CLIProxyAPI/v6/internal/misc"|g' {} +
find internal/tui -type f -name "*.go" -exec sed -i '' 's|"github.com/kooshapari/cliproxyapi-plusplus/v6/pkg/llmproxy/util"|"github.com/router-for-me/CLIProxyAPI/v6/internal/util"|g' {} +
find internal/tui -type f -name "*.go" -exec sed -i '' 's|"github.com/kooshapari/cliproxyapi-plusplus/v6/pkg/llmproxy/auth/kiro"|"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/kiro"|g' {} +
find internal/tui -type f -name "*.go" -exec sed -i '' 's|"github.com/kooshapari/cliproxyapi-plusplus/v6|"github.com/router-for-me/CLIProxyAPI/v6|g' {} +
find internal/tui -type f -name "*.go" -exec sed -i '' 's|"github.com/kooshapari/cliproxyapi-plusplus/|"github.com/router-for-me/CLIProxyAPI/|g' {} +
