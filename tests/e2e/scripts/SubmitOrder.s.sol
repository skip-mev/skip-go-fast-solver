// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Script, console} from "forge-std/Script.sol";

import {FastTransferGateway, FastTransferOrder} from "../src/FastTransferGateway.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {TypeCasts} from "../src/libraries/TypeCasts.sol";

contract DeployScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast();

        address token = address(0xaf88d065e77c8cC2239327C5EDb3A432268e5831); // USDC
        address recipient = address(0x24a9267cE9e0a8F4467B584FDDa12baf1Df772B5);
        uint256 amount = 5_000000;
        uint32 destinationDomain = 8453;
        bytes memory data = bytes("");
        FastTransferGateway gateway = FastTransferGateway(0x83eFe03da48cF12a258c5bb210097E8b0aB2F61F);

        IERC20(token).approve(address(gateway), amount);

        gateway.submitOrder(
            TypeCasts.addressToBytes32(msg.sender),
            TypeCasts.addressToBytes32(recipient),
            amount,
            amount,
            destinationDomain,
            block.timestamp + 1 days,
            data
        );

        vm.stopBroadcast();
    }
}
