# icmp-go-agent
poke ICMP checker agent 

## Configuration

icmp-go-agent will look for configuration files in this order:

- /etc/poke-icmp-agent/
- $HOME/.poke-icmp-agent
- . (current folder)

You can override every configuration entry with environement variables.
Ex:

```sh
export POKE_ICMP_AGENT_HOST=1.icmp-checker.poke.io
```

Needed configuration:

```yaml
host: 1.icmp-checker.poke.io
zone: gra
timeout: 1s
kafka:
  brokers:
    - kafka.poke.io:9092
  user: kafkaUser
  password: kafkaPassword
  topics:
    - check-icmp
```

# Build

Local version:

```sh
make init
make dep
make build
```

Distribution version:

```sh
make dist
```