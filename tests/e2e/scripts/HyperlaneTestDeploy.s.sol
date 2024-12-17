pragma solidity >=0.8.25 <0.9.0;

import { stdJson } from "forge-std/StdJson.sol";
import { Script } from "forge-std/Script.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";
import { Address } from "@openzeppelin/contracts/utils/Address.sol";
import { MockHyperlaneEnvironment } from "@hyperlane-xyz/mock/MockHyperlaneEnvironment.sol";
import { TestMerkle } from "@hyperlane-xyz/test/TestMerkle.sol";
import { ValidatorAnnounce } from "@hyperlane-xyz/isms/multisig/ValidatorAnnounce.sol";

contract HyperlaneTestDeploy is Script {
    using stdJson for string;
    using Address for address;

    function run() public returns (string memory) {
        vm.startBroadcast();

        // Deploy mock Hyperlane environment
        MockHyperlaneEnvironment env = new MockHyperlaneEnvironment(31337, 42161); // Local and Arbitrum domains

        // Deploy additional Hyperlane contracts
        TestMerkle merkleHook = new TestMerkle();
        ValidatorAnnounce validatorAnnounce = new ValidatorAnnounce(address(env.mailboxes(31337)));
        
        vm.stopBroadcast();

        // Create JSON output with Hyperlane contract addresses
        string memory json = "{";
        json = _appendField(json, "mailbox", address(env.mailboxes(31337)));
        json = _appendField(json, "ism", address(env.isms(31337)));
        json = _appendField(json, "merkleHook", address(merkleHook));
        json = _appendField(json, "validatorAnnounce", address(validatorAnnounce));
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