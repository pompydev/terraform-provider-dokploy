# Terraform Provider for Dokploy

This is my personal fork of [ahmedali6/terraform-provider-dokploy][upstream]
(hint: you can read the original README there) created to fix some bugs and
add some features. I do not plan on maintaining it so it is not published to
terraform registry. I honestly wouldn't recommend using it but installation
instruction is provided regardless.

## Installation

Not using the `.tfrc` method because vscode LSP can't pick up new properties.

1. Build

   ```shell
   go build -trimpath -ldflags="-s -w -X main.version=0.7.0" -o terraform-provider-dokploy .
   ```

2. Move binary

   ```shell
   cp terraform-provider-dokploy ~/.terraform.d/plugins/registry.terraform.io/ahmedali6/dokploy/0.7.0/linux_amd64/terraform-provider-dokploy_v0.7.0_x5
   ```

3. Update `~/.terraformrc` (replace `HOME_DIRECTORY_ABSOLUTE_PATH` with `/home/whatever`)

   ```
   provider_installation {
     filesystem_mirror {
       path    = "HOME_DIRECTORY_ABSOLUTE_PATH/.terraform.d/plugins"
       include = ["registry.terraform.io/ahmedali6/dokploy"]
     }

     # For all other providers, install them directly from their origin provider
     # registries as normal.
     direct {
       exclude = ["registry.terraform.io/ahmedali6/dokploy"]
     }
   }
   ```

4. Update `.terraform.lock.hcl` (run it from your terraform project)
   ```shell
   terraform providers lock -fs-mirror="$HOME/.terraform.d/plugins" registry.terraform.io/ahmedali6/dokploy
   ```

[upstream]: https://github.com/ahmedali6/terraform-provider-dokploy
