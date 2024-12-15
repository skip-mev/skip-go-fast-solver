pragma solidity >=0.8.25 <0.9.0;

import { stdJson } from "forge-std/StdJson.sol";
import { Script } from "forge-std/Script.sol";
import { TestERC20 } from "./TestERC20.sol";
import { ICS20Lib } from "./ICS20Lib.sol";
import { FastTransferGateway } from "./FastTransferGateway.sol";
import { ERC1967Proxy } from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";
import {Strings} from "@openzeppelin/contracts/utils/Strings.sol";

contract BaseTestDeploy is Script {
    using stdJson for string;

    string public constant E2E_FAUCET = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266";
    address public constant MOCK_PERMIT2 = 0x0000000000000000000000000000000000000010;

    function run() public returns (string memory) {
        address deployerAddress = msg.sender;
        
        vm.startBroadcast();

        // Deploy ERC20
        TestERC20 erc20 = new TestERC20();

        // Deploy implementation
        FastTransferGateway gatewayImpl = new FastTransferGateway();

        // Prepare initialization data
        bytes memory initData = abi.encodeWithSelector(
            FastTransferGateway.initialize.selector,
            31337,              // Local domain (Anvil chain ID)
            deployerAddress,    // Owner
            address(erc20),     // Token
            address(0),         // Will be set after Hyperlane deployment
            address(0),         // Will be set after Hyperlane deployment
            MOCK_PERMIT2       // Permit2
        );

        // Deploy proxy with initialization data
        ERC1967Proxy gatewayProxy = new ERC1967Proxy(
            address(gatewayImpl),
            initData
        );

        (address addr, bool ok) = ICS20Lib.hexStringToAddress(E2E_FAUCET);
        require(ok, "invalid address");

        erc20.mint(addr, 1_000_000_000_000_000_000);
        
        vm.stopBroadcast();

        // Create JSON output with base contract addresses
        string memory json = "{";
        json = _appendField(json, "erc20", address(erc20));
        json = _appendField(json, "fastTransferGateway", address(gatewayProxy));
        json = string.concat(json, "}");

        return json;
    }

    function _appendField(string memory json, string memory key, address value) internal pure returns (string memory) {
        if (bytes(json).length > 1) {
            json = string.concat(json, ",");
        }
        return string.concat(json, '"', key, '":"', _addressToString(value), '"');
    }

    function _addressToString(address addr) internal pure returns (string memory) {
        return Strings.toHexString(uint160(addr), 20);
    }
}