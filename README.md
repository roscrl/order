# order

Keep YAML and JSON files organised by checking their property order against a JSON schema.

Use `order` if:

- You have many contributors to a repository and want to enforce style consistency
- You want to reduce mental effort in code reviews with a predictable pattern.
- You want your YAML or JSON files to stay neat and consistent.

## Installation

```bash
go get github.com/roscrl/order
```

## Usage

```go
package main

import (
    "fmt"
    "github.com/roscrl/order"
)

func main() {
    // Check if YAML file conforms to property order defined in JSON schema
    err := order.Lint("config.yaml", "schema.json")
    if err != nil {
        return fmt.Errorf("linting properties order against json schema: %v", err)
    }
}
```

## How It Works

`order` looks at the properties list in your JSON schema and makes sure your YAML or JSON file follows the same order.

Properties need to match the schema's order (e.g., "name" before "version").

## Examples

### JSON Schema Example

```json
{
  "properties": {
    "name": { "type": "string" },
    "version": { "type": "string" },
    "description": { "type": "string" },
    "dependencies": {
      "properties": {
        "production": { "type": "object" },
        "development": { "type": "object" }
      }
    },
    "scripts": { "type": "object" }
  }
}
```

### Good YAML

```yaml
name: my-package
version: 1.0.0
description: A sample package
dependencies:
  production:
    some-lib: ^1.0.0
  development:
    test-lib: ^2.0.0
scripts:
  test: jest
```

### Bad YAML (Wrong Order)

```yaml
version: 1.0.0  # Should come after 'name'
name: my-package
description: A sample package
dependencies:
  development:  # Should come after 'production'
    test-lib: ^2.0.0
  production:
    some-lib: ^1.0.0
scripts:
  test: jest
```

This causes an error: "version" is out of order; it should come after "name".

See [examples/main.go](examples/main.go) for a complete example.

## What It Doesn't Do

- It doesn't check types or required fields—just the order.
- It only uses the properties part of the schema.