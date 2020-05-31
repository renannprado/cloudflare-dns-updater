extern crate pnet;
extern crate reqwest;

use std::{env, fs, time};
use std::borrow::Borrow;
use std::collections::HashMap;
use std::fs::File;
use std::io::{Read, Write};
use std::ops::Index;
use std::thread::sleep;

use pnet::datalink;
use pnet::datalink::NetworkInterface;
use pnet::ipnetwork::IpNetwork;

use serde::{Serialize, Deserialize};

fn main() -> std::io::Result<()> {

    let args: Vec<String> = env::args().collect();
    // 1 - interface
    // 2 - cloudflare e-mail
    // 3 - cloudflare api token
    // 3 - cloudflare zone id
    // 4 - last IP file
    // 5 - seconds between cycles
    let iface_prefix = args.index(1);
    let cloudflare_email = args.index(2);
    let cloudflare_api_token = args.index(3);
    let cloudflare_dns_zone = args.index(3);
    let cloudflare_ = args.index(3);
    let cloudflare_dns = args.index(3);
    let last_ip_file = args.index(4);
    let wait_between_cycles = args.index(5).parse::<u64>().unwrap();
    let next_run = time::Duration::from_secs(wait_between_cycles);

    loop {
        let all_ifaces = datalink::interfaces();

        let ifaces = all_ifaces.into_iter().filter(|i|
            i.name.starts_with(iface_prefix)
        // ).collect::<Vec<NetworkInterface>>();
        ).collect::<Vec<NetworkInterface>>();
        let iface = ifaces.index(0);

        println!("checking interface {}", iface);

        let ipv6vec = iface.ips.clone().into_iter().filter( | ip |
            ip.is_ipv6()
        ).collect::<Vec<IpNetwork>>();
        let current_ipv6 = ipv6vec.index(0).ip().to_string();

        println!("{}", current_ipv6);

        // let mut file = File::open(last_ip_file);
        match fs::read_to_string(last_ip_file) {
            Ok(old_ip) => {
                if old_ip == current_ipv6 {
                    println!("IP address didn't change");
                } else {
                    println!("ip changed from {} to {}", old_ip, current_ipv6);
                    let mut file = File::create(last_ip_file)?;
                    file.write_all(current_ipv6.as_bytes())?;
                }
            }
            Err(err) => {
                let mut file = File::create(last_ip_file)?;
                file.write_all(current_ipv6.as_bytes())?;
            }
        }

        println!("sleeping for {:?}", next_run);
        sleep(next_run);
    }

    // let mut file = File::create(last_ip_file)?;
    // file.write_all(b"Hello, world!")?;

    // println!("{:?}", x)
    //
    //
    // for iface in datalink::interfaces() {
    //     println!("{:?}", iface.ips);
    // }
    Ok(())
}

fn update_cloudflare_dns(email: String, api_token: String, dns_zone: String, dns: String, ip: String) {
    // curl -X PUT "https://api.cloudflare.com/client/v4/zones/023e105f4ecef8ad9ca31a8372d0c353/dns_records/372e67954025e0ba6aaa6d586b9e0b59" \
    // -H "X-Auth-Email: user@example.com" \
    // -H "X-Auth-Key: c2547eb745079dac9320b638f5e225cf483cc5cfdda41" \
    // -H "Content-Type: application/json" \
    // --data '{"type":"A","name":"example.com","content":"127.0.0.1","ttl":120,"proxied":false}'
    // #[(Serialize)]
    #[derive(Serialize, Debug)]
    enum Value {
        Int(i32),
        String(String)
    }

    let mut payload = HashMap::new();
    payload.insert("ttl", Value::Int(300));
    payload.insert("type", Value::String("AAAA".to_string()));
    payload.insert("name", Value::String(dns));
    payload.insert("content", Value::String(ip));

    let client = reqwest::blocking::Client::new();
    let res = client.put("https://api.cloudflare.com/client/v4/zones/{}/dns_records/{}")
        .json(&payload)
        .send();

    println!("{:?}", res);
}