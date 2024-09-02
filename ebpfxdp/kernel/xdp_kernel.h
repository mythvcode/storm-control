#include <linux/if_ether.h>
#include <stdbool.h>

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

struct packet_counter {
    traffic_desc  broadcast;
    traffic_desc  ipv4_mcast;
    traffic_desc  ipv6_mcast;
    traffic_desc  other_mcast;
};

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

bool compare_mac_prefix(const unsigned char compare[],
                        unsigned char mac_address[ETH_ALEN],
                        unsigned int len) {
        
    for (int i = 0; i < len; i++) {
        if (compare[i] != mac_address[i]){
            return false;
        }
    }
    return true;
}

bool is_ipv4_mcast(unsigned char mac_address[ETH_ALEN]){
    return compare_mac_prefix(IPV4_MAC_PREFIX, mac_address, 3);
}


bool is_ipv6_mcast(unsigned char mac_address[ETH_ALEN]){
    return compare_mac_prefix(IPV6_MAC_PREFIX, mac_address, 2);
}

bool is_other_mcast(unsigned char mac_address[ETH_ALEN]){
    const char mask = 0x01;
    return (mac_address[0] & mask) == 0x01;
}

bool is_broadcast(unsigned char mac_address[ETH_ALEN]){
    return compare_mac_prefix(BROADCAST, mac_address, ETH_ALEN);
}