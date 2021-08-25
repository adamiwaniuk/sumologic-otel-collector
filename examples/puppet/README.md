# Installation of Sumo Logic Distro of OpenTelemetry Collector with puppet

This [puppet][puppet] manifest along with [module](modules/install_otel_collector/) will install Sumo Logic Distro of [OpenTelemetry Collector][otc_link].

## Configuration

- Prepare [configuration](../../docs/Configuration.md) for Sumo Logic Distro of of OpenTelemetry Collector and
  save it in [files](modules/install_otel_collector/files/) directory for `instal_otel_collector` module as `config.yaml`.
- Adjust settings for Systemd Service in [system_service](modules/install_otel_collector/files/systemd_service).

## Test on Vagrant

Vagrant environment has puppet agent and puppet server installed on single host.
Example puppet manifest and module are mounted to Vagrant virtual machine:

- [modules/](modules/)  is mounted to `/etc/puppetlabs/code/environments/production/modules/`
- [manifests/](manifests/) is mounted to `/etc/puppetlabs/code/environments/production/manifests/`

To install Sumo Logic Distro of OpenTelemetry Collector with puppet on Vagrant virtual machine:

- Prepare configuration using steps described in [Configuration](#configuration)
- From main directory of this repository start virtual machine:

  ```bash
  vagrant up
  ```

- Connect to virtual machine:

  ```bash
  vagrant ssh
  ```

- Pull configuration for puppet agent:

  ```bash
  sudo puppet agent --test --waitforcert 60
  ```

- In another terminal window for Vagrant virtual machine, sign the certificate:

  ```bash
  sudo puppetserver ca sign --certname agent.example.com
  ```

- See that puppet agent pulls configuration from puppet server.
- Verify installation:

  ```bash
  sudo systemctl status otelcol-sumo
  sudo journalctl -u otelcol-sumo
  ```

[otc_link]: https://github.com/open-telemetry/opentelemetry-collector
[puppet]: https://puppet.com/
