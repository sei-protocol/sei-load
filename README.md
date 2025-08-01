#  seiload CLI
[![Tests](https://github.com/sei-protocol/sei-load/actions/workflows/build-and-test.yml/badge.svg)](https://github.com/sei-protocol/sei-load/actions/workflows/build-and-test.yml)

A comprehensive load testing tool for Sei Chain that supports both contract and non-contract scenarios with advanced worker management, receipt tracking, and block monitoring capabilities.

## Features

- **Multi-Endpoint Support**: Distribute load across multiple RPC endpoints
- **Scenario-Based Testing**: Weighted scenario selection with factory patterns
- **Worker Pool Management**: Configurable worker pools per endpoint
- **Receipt Tracking**: Optional transaction receipt monitoring
- **Block Monitoring**: Real-time block statistics collection
- **Account Management**: Automatic account generation and pooling
- **Rate Limiting**: Configurable transactions per second (TPS) limits
- **Dry Run Mode**: Test configurations without sending actual transactions

## Installation

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd sei-load

# Build the CLI
make build-cli

# Install globally (optional)
make install
```

### Prerequisites

- Go 1.23.7 or later
- Access to Sei Chain RPC endpoints

## Quick Start

1. **Create a configuration file** (see [Configuration](#configuration) section):

```bash
cp profiles/local.json my-config.json
# Edit my-config.json with your settings
```

2. **Run a basic load test**:

```bash
./build/seiload --config my-config.json --workers 5 --tps 100
```

3. **Run with receipt and block tracking**:

```bash
./build/seiload --config my-config.json --workers 10 --track-receipts --track-blocks
```

## Configuration

### Configuration File Structure

```json
{
  "chain_id": 713714,
  "endpoints": [
    "http://127.0.0.1:8545",
    "http://127.0.0.1:8546"
  ],
  "accounts": {
    "accounts": 5000,
    "new_account_rate": 0.1
  },
  "scenarios": [
    {
      "name": "ERC20",
      "weight": 2
    },
    {
      "name": "EVMTransfer",
      "weight": 1
    }
  ]
}
```

### Configuration Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `chain_id` | Blockchain chain ID | Required |
| `endpoints` | Array of RPC endpoint URLs | Required |
| `accounts.accounts` | Number of accounts to generate | 1000 |
| `accounts.new_account_rate` | Rate of new account creation | 0.0 |
| `scenarios` | Array of test scenarios with weights | Required |

### Available Scenarios

- **ERC20**: ERC-20 token operations
- **EVMTransfer**: Direct ETH transfers
- **ERC20Noop**: No-operation ERC-20 calls
- **ERC20Conflict**: Conflicting ERC-20 transactions
- **ERC721**: NFT operations

## Command Line Options

```bash
seiload [flags]

Flags:
  -c, --config string          Path to configuration file (required)
  -w, --workers int           Number of workers per endpoint (default 1)
  -t, --tps float             Transactions per second limit (0 = no limit)
  -s, --stats-interval duration  Statistics logging interval (default 10s)
  -b, --buffer-size int       Buffer size per worker (default 1000)
      --track-receipts        Enable transaction receipt tracking
      --track-blocks          Enable block statistics collection
      --dry-run              Test mode without sending transactions
      --debug                Enable detailed request logging
      --prewarm              Prewarm accounts with self-transactions
  -h, --help                 Show help information
```

## Core Components

### Workers

Workers are the core execution units that process and send transactions to blockchain endpoints.

#### How Workers Function

- **Worker Pools**: Each endpoint gets its own pool of workers
- **Concurrent Processing**: Multiple workers per endpoint process transactions in parallel
- **Load Distribution**: Transactions are distributed across workers using round-robin
- **Connection Pooling**: Each worker maintains optimized HTTP connections

#### Worker Configuration

```bash
# Single worker per endpoint
seiload --config config.json --workers 1

# 5 workers per endpoint (recommended for high throughput)
seiload --config config.json --workers 5

# High-performance setup
seiload --config config.json --workers 10 --buffer-size 2000
```

#### Worker Architecture

```
Dispatcher → ShardedSender → Worker Pool → RPC Endpoints
                ↓
            [Worker 1] → Endpoint 1
            [Worker 2] → Endpoint 1
            [Worker 3] → Endpoint 2
            [Worker 4] → Endpoint 2
```

### Track Receipts

Receipt tracking monitors transaction execution status and provides detailed success/failure metrics.

#### What Receipt Tracking Does

- **Transaction Monitoring**: Polls for transaction receipts after submission
- **Status Verification**: Confirms transaction success or failure
- **Gas Usage Tracking**: Records actual gas consumption
- **Timeout Handling**: Manages receipt polling with configurable timeouts

#### Enabling Receipt Tracking

```bash
# Enable receipt tracking
seiload --config config.json --track-receipts

# With debug output for detailed receipt information
seiload --config config.json --track-receipts --debug
```

#### Receipt Tracking Behavior

- **Polling Interval**: Checks every 100ms for receipts
- **Timeout**: 10-second timeout per transaction
- **Status Reporting**: Logs success/failure with gas usage
- **Performance Impact**: Adds overhead but provides valuable metrics

#### Example Output with Receipt Tracking

```
✅ tx ERC20, 0x1234...abcd, gas=21000 succeeded
❌ tx 0x5678...efgh failed
❌ timeout waiting for receipt for tx 0x9abc...def0
```

### Track Blocks

Block tracking provides real-time blockchain performance metrics by monitoring block production.

#### What Block Tracking Does

- **Block Monitoring**: Subscribes to new block headers via WebSocket
- **Performance Metrics**: Calculates block times and gas usage statistics
- **Statistical Analysis**: Provides P50, P99, and maximum values
- **Window-Based Reporting**: Tracks both cumulative and recent performance

#### Enabling Block Tracking

```bash
# Enable block tracking
seiload --config config.json --track-blocks

# Combined with receipt tracking for comprehensive monitoring
seiload --config config.json --track-receipts --track-blocks
```

#### Block Statistics Collected

- **Block Times**: Time between consecutive blocks
- **Gas Usage**: Gas consumed per block
- **Block Numbers**: Highest block number observed
- **Percentile Analysis**: P50, P99 performance metrics

#### Example Block Statistics Output

```
[15:04:05] Blocks: #12345 | Times: P50=2.1s P99=4.2s Max=5.8s | Gas: P50=8.2M P99=15.1M Max=20.0M | Samples: 150
```

#### Block Tracking Architecture

```
WebSocket Connection → Block Headers → Statistics Engine → Periodic Reports
        ↓
    ws://endpoint:8546 → BlockCollector → Metrics Calculation → Console Output
```

## Usage Examples

### Basic Load Test

```bash
# Simple load test with 3 workers
seiload --config profiles/local.json --workers 3
```

### High-Throughput Testing

```bash
# High-performance setup with rate limiting
seiload --config my-config.json \
  --workers 10 \
  --tps 500 \
  --buffer-size 2000 \
  --stats-interval 5s
```

### Comprehensive Monitoring

```bash
# Full monitoring with receipts and blocks
seiload --config my-config.json \
  --workers 8 \
  --track-receipts \
  --track-blocks \
  --debug
```

### Development and Testing

```bash
# Dry run to test configuration
seiload --config my-config.json --dry-run

# Debug mode with detailed logging
seiload --config my-config.json --debug --workers 2
```

### Production Load Testing

```bash
# Production-ready configuration
seiload --config production.json \
  --workers 15 \
  --tps 1000 \
  --track-receipts \
  --track-blocks \
  --stats-interval 10s \
  --buffer-size 5000
```

## Performance Tuning

### Worker Optimization

- **Start Small**: Begin with 1-3 workers per endpoint
- **Scale Gradually**: Increase workers based on endpoint capacity
- **Monitor Resources**: Watch CPU and memory usage
- **Optimal Range**: 5-15 workers per endpoint for most scenarios

### Buffer Size Tuning

- **Low Latency**: Smaller buffers (500-1000) for quick feedback
- **High Throughput**: Larger buffers (2000-5000) for sustained load
- **Memory Consideration**: Larger buffers consume more memory

### Rate Limiting

```bash
# Conservative rate limiting
seiload --config config.json --tps 100

# Aggressive testing (use with caution)
seiload --config config.json --tps 2000

# No rate limiting (maximum throughput)
seiload --config config.json --tps 0
```

## Monitoring and Statistics

### Real-Time Statistics

The tool provides comprehensive real-time statistics:

- **Transaction Metrics**: Total transactions, TPS, success rates
- **Latency Analysis**: P50, P99, maximum response times
- **Endpoint Performance**: Per-endpoint statistics
- **Block Performance**: Block times and gas usage (with `--track-blocks`)

### Example Statistics Output

```
[15:04:05] Total: 15,420 txs | Avg Latency: 145ms | P50: 120ms | P99: 280ms | Max: 450ms
[15:04:05] ep1: 7,710 txs (385.5 TPS) | Latency: 142ms | ep2: 7,710 txs (385.5 TPS) | Latency: 148ms
[15:04:05] Blocks: #12345 | Times: P50=2.1s P99=4.2s Max=5.8s | Gas: P50=8.2M P99=15.1M Max=20.0M
```

## Troubleshooting

### Common Issues

1. **Connection Errors**
   ```bash
   # Verify endpoint connectivity
   curl -X POST -H "Content-Type: application/json" \
     --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
     http://your-endpoint:8545
   ```

2. **High Memory Usage**
   - Reduce buffer size: `--buffer-size 500`
   - Decrease worker count: `--workers 3`
   - Disable receipt tracking if not needed

3. **Low Performance**
   - Increase worker count: `--workers 10`
   - Increase buffer size: `--buffer-size 2000`
   - Remove rate limiting: `--tps 0`

### Debug Mode

Enable debug mode for detailed troubleshooting:

```bash
seiload --config config.json --debug --dry-run
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

[Add your license information here]

## Support

For issues and questions:
- Create an issue in the repository
- Check existing documentation
- Review configuration examples in the `profiles/` directory
