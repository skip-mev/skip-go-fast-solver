pragma solidity >=0.8.25 <0.9.0;

import { stdJson } from "forge-std/StdJson.sol";
import { Script } from "forge-std/Script.sol";
import { TestERC20 } from "./TestERC20.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";
import { ICS20Lib } from "./ICS20Lib.sol";
import { FastTransferGateway } from "./FastTransferGateway.sol";
import { ERC1967Proxy } from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";
import { MockHyperlaneEnvironment } from "@hyperlane-xyz/mock/MockHyperlaneEnvironment.sol";
import { TestMerkle } from "@hyperlane-xyz/test/TestMerkle.sol";
import { ValidatorAnnounce } from "@hyperlane-xyz/ValidatorAnnounce.sol";

contract MockE2ETestDeploy is Script {
    using stdJson for string;

    string public constant E2E_FAUCET = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266";
    address public constant MOCK_PERMIT2 = 0x0000000000000000000000000000000000000010;

    function run() public returns (string memory) {
        address deployerAddress = msg.sender;
        
        vm.startBroadcast();

        // Deploy mock Hyperlane environment
        MockHyperlaneEnvironment env = new MockHyperlaneEnvironment(31337, 42161); // Local and Arbitrum domains

        // Deploy additional Hyperlane contracts
        TestMerkle merkleHook = new TestMerkle();
        ValidatorAnnounce validatorAnnounce = new ValidatorAnnounce(address(env.mailboxes(31337)));
        
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
            address(env.mailboxes(31337)),  // Mailbox
            address(env.isms(31337)),       // InterchainSecurityModule
            MOCK_PERMIT2       // Permit2
        );

        // Deploy proxy with initialization data
        ERC1967Proxy gatewayProxy = new ERC1967Proxy(
            address(gatewayImpl),
            initData
        );

        // Cast proxy to FastTransferGateway for easier interaction
        FastTransferGateway ftg = FastTransferGateway(address(gatewayProxy));

        // Verify initialization
        require(ftg.owner() == deployerAddress, "Initialization failed: owner not set");
        
        (address addr, bool ok) = ICS20Lib.hexStringToAddress(E2E_FAUCET);
        require(ok, "invalid address");

        erc20.mint(addr, 1_000_000_000_000_000_000);
        
        vm.stopBroadcast();

        // Create JSON output with all contract addresses
        string memory json = "{";
        json = _appendField(json, "erc20", address(erc20));
        json = _appendField(json, "fastTransferGateway", address(gatewayProxy));
        json = _appendField(json, "mailbox", address(env.mailboxes(31337)));
        json = _appendField(json, "ism", address(env.isms(31337)));
        json = _appendField(json, "merkleHook", address(merkleHook));
        json = _appendField(json, "validatorAnnounce", address(validatorAnnounce));
        json = string.concat(json, "}");

        return json;
    }

    function _appendField(string memory json, string memory key, address value) internal pure returns (string memory) {
        if (bytes(json).length > 1) { // If not first field
            json = string.concat(json, ",");
        }
        return string.concat(json, '"', key, '":"', _addressToString(value), '"');
    }

    function _addressToString(address addr) internal pure returns (string memory) {
        return Strings.toHexString(uint160(addr), 20);
    }
}