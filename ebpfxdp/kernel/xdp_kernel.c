// must be included first
#include <linux/types.h>

#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>
#include <linux/bpf.h>
#include "xdp_kernel.h"

struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_HASH);
    __type(key, __u32);
    __type(value, struct packet_counter);
    __uint(max_entries, CONFIG_MAP_MAX_ELEMENT);
} intf_stats SEC(".maps");


struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u32);
    __type(value, drop_pkt);
    __uint(max_entries, CONFIG_MAP_MAX_ELEMENT);
} drop_intf SEC(".maps");

static __always_inline int proto_is_vlan(__u16 h_proto) {
    return !!(h_proto == bpf_htons(ETH_P_8021Q) ||
              h_proto == bpf_htons(ETH_P_8021AD));
}

// used to find h_proto value in case of vlan tags
// support maximum of two vlans(q-in-q)
// return value in big endian
static __always_inline __be16 get_h_proto(struct ethhdr *eth, void *data_end) {
    __u16 h_proto = eth->h_proto;
    struct vlan_hdr *vlan;
    if (!proto_is_vlan(h_proto)){
        return h_proto;
    }
    if ((void*)(eth + 1) + sizeof(struct vlan_hdr) > data_end){
        return h_proto;
    }
    vlan = (void*)(eth + 1);
    h_proto = vlan->h_vlan_encapsulated_proto;
    if (!proto_is_vlan(vlan->h_vlan_encapsulated_proto)){
        return h_proto;
    }

    if ((void*)(vlan + 1) + sizeof(struct vlan_hdr) > data_end){
        return h_proto;
    }
    // second vlan header(q-in-q)
    vlan = vlan + 1;
    return vlan->h_vlan_encapsulated_proto;
}

static __always_inline int get_xdp_action(traffic_desc *tr_desc, __u32 ifindex, p_type pt){
    drop_pkt *drop_desc = bpf_map_lookup_elem(&drop_intf, &ifindex);
    if (!drop_desc){
        tr_desc->passed++;
        return XDP_PASS;
    }
    if (drop_desc->broadcast && pt == Broadcast){
        tr_desc->dropped++;
        return XDP_DROP;
    } else if (drop_desc->ipv4_mcast && pt == IPv4MCast){
        tr_desc->dropped++;
        return XDP_DROP;
    } else if (drop_desc->ipv6_mcast && pt == IPv6MCast){
        tr_desc->dropped++;
        return XDP_DROP;
    } else if (drop_desc->other_mcast && pt == GenericMCast){
        tr_desc->dropped++;
        return XDP_DROP;
    }
    tr_desc->passed++;
    return XDP_PASS;
}

// calculate packets and return xdp_action
static __always_inline int calculate_pkt(struct ethhdr *eth, __u32 ifindex, __u16 h_proto) {
    struct packet_counter *count_s;
    count_s = bpf_map_lookup_elem(&intf_stats, &ifindex);

    if (!count_s){
        return XDP_PASS;
    }
    if (is_broadcast(eth->h_dest)){
        return get_xdp_action(&(count_s->broadcast), ifindex, Broadcast);

    } else if (is_ipv4_mcast(eth->h_dest) && h_proto==ETH_P_IP){
        return get_xdp_action(&(count_s->ipv4_mcast), ifindex, IPv4MCast);

    } else if (is_ipv6_mcast(eth->h_dest) && h_proto==ETH_P_IPV6){
        return get_xdp_action(&(count_s->ipv6_mcast), ifindex, IPv6MCast);

    } else if (is_other_mcast(eth->h_dest)){
        return get_xdp_action(&(count_s->other_mcast), ifindex, IPv4MCast);
    }

    return XDP_PASS;
}

SEC("xdp")
int storm_control(struct xdp_md *ctx)
{
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;
    struct ethhdr *eth = data;

    if (data + sizeof(struct ethhdr) > data_end){
        return XDP_PASS;
    }
    return calculate_pkt(eth, ctx->ingress_ifindex, bpf_ntohs(get_h_proto(eth, data_end)));

}
