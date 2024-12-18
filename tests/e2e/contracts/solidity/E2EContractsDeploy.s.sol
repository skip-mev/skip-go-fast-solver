pragma solidity >=0.8.25 <0.9.0;

import { stdJson } from "forge-std/StdJson.sol";
import { Script } from "forge-std/Script.sol";
import { TestERC20 } from "./TestERC20.sol";
import { FastTransferGateway } from "./FastTransferGateway.sol";
import { ERC1967Proxy } from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";

contract E2EContractsDeploy is Script {
    using stdJson for string;

    string public constant E2E_FAUCET = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266";
    address public constant MOCK_PERMIT2 = 0x0000000000000000000000000000000000000010;

    function run() public returns (string memory) {
        address deployerAddress = msg.sender;
        
        vm.startBroadcast();

        TestERC20 erc20 = new TestERC20();

        FastTransferGateway gatewayImpl = new FastTransferGateway();

        bytes memory initData = abi.encodeWithSelector(
            FastTransferGateway.initialize.selector,
            31337,
            deployerAddress,
            address(erc20),
            address(0),         // Will be set after Hyperlane deployment
            address(0),         // Will be set after Hyperlane deployment
            MOCK_PERMIT2
        );

        ERC1967Proxy gatewayProxy = new ERC1967Proxy(
            address(gatewayImpl),
            initData
        );

        (address addr, bool ok) = hexStringToAddress(E2E_FAUCET);
        require(ok, "invalid address");

        erc20.mint(addr, 1_000_000_000_000_000_000);
        
        vm.stopBroadcast();

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

    function hexStringToAddress(string memory addrHexString) internal pure returns (address, bool) {
        bytes memory addrBytes = bytes(addrHexString);
        if (addrBytes.length != 42) {
            return (address(0), false);
        } else if (addrBytes[0] != "0" || addrBytes[1] != "x") {
            return (address(0), false);
        }
        uint256 addr = 0;
        unchecked {
            for (uint256 i = 2; i < 42; i++) {
                uint256 c = uint256(uint8(addrBytes[i]));
                if (c >= 48 && c <= 57) {
                    addr = addr * 16 + (c - 48);
                } else if (c >= 97 && c <= 102) {
                    addr = addr * 16 + (c - 87);
                } else if (c >= 65 && c <= 70) {
                    addr = addr * 16 + (c - 55);
                } else {
                    return (address(0), false);
                }
            }
        }
        return (address(uint160(addr)), true);
    }
}