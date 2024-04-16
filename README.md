# Eachit

Eachit is a command-line tool that automates the process of removing specified containers and building HCL files using Packer. It provides a convenient way to manage and build infrastructure as code.

## Features

- Remove specified containers using the `incus` command
- Build HCL files using Packer
- Send build notifications via the Ntfy channel
- Retry logic for container removal
- Logging of build duration and status

## Usage

```bash
eachit run [flags]
```

### Flags

- `--destroy-containers`: List of container names to remove (default: `["mythai"]`)
- `--exclude-hcls`: List of HCL files to exclude from building
- `--hcl`: List of HCL files to build (overrides default build list)
- `--ntfy-channel`: Ntfy channel for sending build notifications

## Example

```bash
eachit run --destroy-containers mythai,mycontainer --exclude-hcls example.hcl --hcl main.hcl --ntfy-channel myproject
```

This command will:
- Remove the containers named `mythai` and `mycontainer`
- Exclude the `example.hcl` file from building
- Build the `main.hcl` file using Packer
- Send build notifications to the Ntfy channel `https://ntfy.sh/myproject`

## Dependencies

Eachit relies on the [Packer Plugin for Incus](https://github.com/bketelsen/packer-plugin-incus?tab=readme-ov-file#packer-plugin-incus) to manage containers and build infrastructure. Make sure you have the plugin installed and properly configured before using Eachit.

## License

Eachit is open-source software licensed under the [MIT License](https://opensource.org/licenses/MIT).