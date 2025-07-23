# MongoDB Connection Issue Solutions

## Problem Description
The application fails to connect to MongoDB Atlas with the error:
```
Failed to connect to MongoDB: error parsing uri: see https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo#hdr-Potential_DNS_Issues: lookup eastcluster.qlszngk.mongodb.net on 8.8.8.8:53: cannot unmarshal DNS message
```

This error occurs because the DNS server (8.8.8.8) cannot resolve the MongoDB Atlas SRV record for `eastcluster.qlszngk.mongodb.net`.

## Root Cause Analysis
1. **DNS Resolution Issue**: The sandbox environment's DNS configuration cannot resolve MongoDB Atlas SRV records
2. **SRV Record Dependency**: MongoDB+srv:// connection strings rely on DNS SRV and TXT records
3. **Network Environment**: The current environment may have DNS restrictions or firewall rules

## Solutions

### Solution 1: Use Alternative DNS Servers (Recommended)
Modify the system DNS configuration to use DNS servers that can resolve MongoDB Atlas domains.

**Steps:**
1. Update `/etc/resolv.conf`:
```bash
sudo echo "nameserver 1.1.1.1" > /etc/resolv.conf
sudo echo "nameserver 8.8.4.4" >> /etc/resolv.conf
```

2. Test DNS resolution:
```bash
nslookup eastcluster.qlszngk.mongodb.net
```

3. Use the original connection string:
```
MONGODB_URI=mongodb+srv://todo:wH0YRnD6flBtZwkv@eastcluster.qlszngk.mongodb.net/?retryWrites=true&w=majority&appName=EastCluster
```

### Solution 2: Convert to Standard Connection String
Manually resolve the SRV record and use a standard MongoDB connection string.

**Steps:**
1. Resolve SRV records manually (from a machine with working DNS):
```bash
dig SRV _mongodb._tcp.eastcluster.qlszngk.mongodb.net
```

2. Use the resolved IPs in a standard connection string:
```
MONGODB_URI=mongodb://todo:wH0YRnD6flBtZwkv@[IP1]:27017,[IP2]:27017,[IP3]:27017/?ssl=true&replicaSet=[REPLICA_SET_NAME]&authSource=admin&retryWrites=true&w=majority
```

### Solution 3: Enhanced Connection Configuration
Improve the MongoDB client configuration with better timeout and retry settings.

**Implementation:**
The `internal/database/clients.go` file has been updated with:
- Increased connection timeouts (30 seconds)
- Better connection pool settings
- Enhanced error handling
- Longer ping timeout for initial connection verification

### Solution 4: Environment-Specific Configuration
Create different environment configurations for different deployment environments.

**Files provided:**
- `.env.solution1` - Original SRV connection (use with DNS fix)
- `.env.solution2` - Template for standard connection string
- `internal/database/clients.go` - Enhanced MongoDB client with better timeouts

## Testing the Solutions

### Test DNS Resolution:
```bash
ping eastcluster.qlszngk.mongodb.net
nslookup eastcluster.qlszngk.mongodb.net
```

### Test Application:
```bash
# Copy the desired solution
cp .env.solution1 .env

# Run the application
go run cmd/main.go
```

## Additional Recommendations

1. **Production Deployment**: Use environment-specific DNS configurations
2. **Connection Monitoring**: Implement connection health checks
3. **Fallback Strategy**: Consider multiple connection strings for redundancy
4. **Security**: Ensure MongoDB credentials are properly secured
5. **Logging**: Enhanced logging has been added to track connection issues

## Dependencies Fixed
The application also had missing dependencies that have been resolved:
- `yt-dlp` - Python package for video downloading
- `aria2` - Download utility

## Files Modified/Created
- `internal/database/clients.go` - Enhanced with better connection handling
- `.env.solution1` - DNS-based solution
- `.env.solution2` - Standard connection string template
- `MONGODB_CONNECTION_SOLUTIONS.md` - This documentation

## Next Steps
1. Try Solution 1 first (DNS configuration)
2. If DNS cannot be modified, use Solution 2 (standard connection string)
3. Monitor connection stability in production
4. Consider implementing connection retry logic for production use

