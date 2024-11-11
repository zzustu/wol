package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"net"
	"strings"
)

var help = `
环境说明：
	1. 被唤醒主机要支持Wake on Lan功能且已开启该功能
	2. 被唤醒主机在关机时应该是有线连接
	3. 请确保该程序与被唤醒主机在同一局域网中

参数说明：
	-mac 被唤醒主机的MAC地址 (必须输入)
	-nic 指定网卡发送唤醒魔包

使用说明：
	wol -mac 11:22:33:44:55:66 -nic eno1

注意事项：
	1. 路由器设备或有些主机有多张网卡, 如果不指定网卡唤醒魔包可能到达不了被唤醒主机
	2. MAC地址的格式可以是以下几种形式
		11:22:33:44:55:66 或
		11-22-33-44-55-66 或
		11:22-33:44:55-66 或
		112233445566

源码地址:
	https://github.com/zzustu/wol
`

// 网络唤醒魔包技术白皮书地址: https://www.amd.com/content/dam/amd/en/documents/archived-tech-docs/white-papers/20213.pdf
func main() {
	mac := flag.String("mac", "", help)
	nic := flag.String("nic", "", help)
	flag.Parse()
	if len(*mac) == 0 {
		fmt.Printf("%s\n", help)
		return
	}

	hw := strings.Replace(strings.Replace(*mac, ":", "", -1), "-", "", -1)
	if len(hw) != 12 {
		fmt.Printf("MAC: [%s] 输入不正确.\n", *mac)
		return
	}

	macHex, err := hex.DecodeString(hw)
	if err != nil {
		fmt.Printf("MAC: [%s] 输入不正确.\n", *mac)
		return
	}

	// 广播MAC地址 FF:FF:FF:FF:FF:FF
	var bcast = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	var buff bytes.Buffer
	buff.Write(bcast)
	for i := 0; i < 16; i++ {
		buff.Write(macHex)
	}

	// 获得唤醒魔包
	mp := buff.Bytes()
	if len(mp) != 102 {
		fmt.Printf("MAC: [%s] 输入不正确.\n", *mac)
		return
	}

	sendMagicPacket(mp, *nic)
}

// 向指定网卡发送唤醒魔包
func sendMagicPacket(mp []byte, nic string) {
	sender := net.UDPAddr{}
	if len(nic) != 0 {
		ip, err := interfaceIPv4ByName(nic)
		if err != nil {
			fmt.Printf("网卡[%s]错误: %s", nic, err)
			return
		}

		sender.IP = ip
	}

	target := net.UDPAddr{
		IP: net.IPv4bcast,
	}
	conn, err := net.DialUDP("udp", &sender, &target)
	if err != nil {
		fmt.Printf("创建UDP错误：%v", err)
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	_, err = conn.Write(mp)
	if err != nil {
		fmt.Printf("魔包发送失败[%s]", err)
	} else {
		fmt.Printf("魔包发送成功")
	}
}

// 通过网卡名称获取该网卡绑定的IPv4
func interfaceIPv4ByName(nic string) (net.IP, error) {
	inter, err := net.InterfaceByName(nic)
	if err != nil {
		return nil, err
	}

	// 检查网卡是否正在工作
	if (inter.Flags & net.FlagUp) == 0 {
		return nil, errors.New("网卡未工作")
	}

	addrs, err := inter.Addrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ip, ok := addr.(*net.IPNet); ok {
			if ipv4 := ip.IP.To4(); ipv4 != nil {
				return ipv4, nil
			}
		}
	}

	return nil, errors.New("该网卡没有IPv4地址")
}
