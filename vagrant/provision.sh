#!/usr/bin/env bash

export BUILDER_VERSION=0.31.0
export GO_VERSION=1.17

# Install opentelemetry-collector-builder
curl -LJ \
    "https://github.com/open-telemetry/opentelemetry-collector-builder/releases/download/v${BUILDER_VERSION}/opentelemetry-collector-builder_${BUILDER_VERSION}_linux_amd64" \
    -o /usr/local/bin/opentelemetry-collector-builder \
    && chmod +x /usr/local/bin/opentelemetry-collector-builder

sudo apt update -y
sudo apt install -y \
    make \
    gcc \
    python3-pip

# Install Go
curl -LJ "https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o go.linux-amd64.tar.gz \
    && rm -rf /usr/local/go \
    && tar -C /usr/local -xzf go.linux-amd64.tar.gz \
    && rm go.linux-amd64.tar.gz \
    && ln -s /usr/local/go/bin/go /usr/local/bin

# Install ansible
pip3 install ansible

# Add puppet hosts
tee -a /etc/hosts << END
192.168.79.13 puppetserver.example.com puppet
192.168.79.13 agent.example.com
END

# Install puppet server & puppet agent
wget https://apt.puppetlabs.com/puppet6-release-focal.deb
dpkg -i puppet6-release-focal.deb
apt-get update -y
apt-get install puppetserver -y
apt-get install puppet-agent -y

tee /etc/puppetlabs/puppet/puppet.conf << END
[server]
vardir = /opt/puppetlabs/server/data/puppetserver
logdir = /var/log/puppetlabs/puppetserver
rundir = /var/run/puppetlabs/puppetserver
pidfile = /var/run/puppetlabs/puppetserver/puppetserver.pid
codedir = /etc/puppetlabs/code

certname = puppetserver.example.com
server = puppet
dns_alt_names = puppetserver.example.com,puppet,puppet.example.com

[agent]
certname = agent.example.com
server = puppet
END

# Start puppet server
systemctl start puppetserver
systemctl enable puppetserver

# Start puppet agent
systemctl start puppet
systemctl enable puppet

echo 'PATH="$PATH:/opt/puppetlabs/bin/"' >> /etc/profile
sed -i 's#secure_path="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin"#secure_path="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin:/opt/puppetlabs/bin"#g' /etc/sudoers
