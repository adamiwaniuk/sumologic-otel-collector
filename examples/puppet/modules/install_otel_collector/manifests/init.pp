class install_otel_collector {

   $otel_collector_version = "0.0.18"

   $arch = $facts['os']['architecture'] ? {
      'aarch64' => 'arm64',
      'arm64'   => 'arm64',
      default   => 'amd64',
   }

   $os_family = $facts['os']['family'] ? {
      'Darwin' => 'darwin',
      default   => 'linux',
   }

   exec {"download the release binary":
      cwd     => "/usr/local/bin/",
      command => "curl -sLo otelcol-sumo https://github.com/SumoLogic/sumologic-otel-collector/releases/download/v${otel_collector_version}/otelcol-sumo-${otel_collector_version}-${os_family}_${arch}",
      path    => ["/usr/bin", "/usr/sbin"],
   }

   exec {"make otelcol-sumo executable":
      cwd     => "/usr/local/bin/",
      command => "chmod +x otelcol-sumo",
      path    => ["/usr/bin", "/usr/sbin"],
   }

   file {"/etc/otelcol-sumo":
     ensure => 'directory',
   }

   file {"/etc/systemd/system/otelcol-sumo.service":
     source => "puppet:///modules/install_otel_collector/systemd_service",
     mode => "644",
   }

   file {"/etc/otelcol-sumo/config.yaml":
     source => "puppet:///modules/install_otel_collector/config.yaml",
     mode => "644",
   }

   group {"opentelemetry":
      ensure  => "present",
   }

   user {"opentelemetry":
      ensure  => "present",
      groups  => ["opentelemetry"],
   }

   service {"otelcol-sumo":
      ensure => "running",
      enable => true,
   }

   # exec { 'run otelcol-sumo in background':
   #    cwd => '/etc/otelcol-sumo/',
   #    command => '/bin/bash -c "/usr/local/bin/otelcol-sumo --log-level DEBUG --log-profile dev  --config /etc/otelcol-sumo/config.yaml > /var/log/otelcol.log 2>&1 &"',
   #    path    => ['/usr/local/bin/', '/usr/bin', '/usr/sbin'],
   #    logoutput => true,
   #    provider => shell,
   #    user => root,
   # }
}
