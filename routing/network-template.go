package routing

const networkRoutingTemplate = `# ensure that traffic arriving on eth1 leaves again from eth1 to prevent asymetric routing
[Match]
Name=eth1
[Network]
Address={{.ENIAddress}}/{{.ENISubnetSize}}

[RoutingPolicyRule]
Table=2
From={{.ENIAddress}}/32

[Route]
Destination=0.0.0.0/0
Gateway={{.ENIGateway}}
Table=2

[Route]
Destination={{.ENISubnet}}/{{.ENISubnetSize}}
Table=2
Scope=link`
