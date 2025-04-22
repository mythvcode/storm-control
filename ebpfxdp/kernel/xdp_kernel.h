#include <linux/if_ether.h>

#define CONFIG_MAP_MAX_ELEMENT 10000

typedef enum {
    Broadcast,
    IPv4MCast,
    IPv6MCast,
    GenericMCast
} p_type;

const unsigned char BROADCAST[ETH_ALEN] = {0xff, 0xff, 0xff, 0xff, 0xff, 0xff};
const unsigned char IPV4_MAC_PREFIX[3]  = {0x01, 0x00, 0x5e};
const unsigned char IPV6_MAC_PREFIX[2]  = {0x33, 0x33};


typedef struct {
    __u64 passed;
    __u64 dropped;
} traffic_desc;

typedef struct  {
    traffic_desc  broadcast;
    traffic_desc  ipv4_mcast;
    traffic_desc  ipv6_mcast;
    traffic_desc  other_mcast;
} packet_counter;

typedef struct {
    __u8 broadcast;
    __u8 ipv4_mcast;
    __u8 ipv6_mcast;
    __u8 other_mcast;
} drop_pkt;


struct vlan_hdr {
    __be16  h_vlan_TCI;
    __be16  h_vlan_encapsulated_proto;
};

int is_ipv4_mcast(const unsigned char mac_address[ETH_ALEN]){
    return !__builtin_memcmp(IPV4_MAC_PREFIX, mac_address, 3);
}


int is_ipv6_mcast(const unsigned char mac_address[ETH_ALEN]){
    return !__builtin_memcmp(IPV6_MAC_PREFIX, mac_address, 2);
}

int is_other_mcast(const unsigned char mac_address[ETH_ALEN]){
    return (mac_address[0] & 0x01);
}

int is_broadcast(const unsigned char mac_address[ETH_ALEN]){
    return !__builtin_memcmp(BROADCAST, mac_address, ETH_ALEN);
}
