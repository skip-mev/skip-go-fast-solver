// SPDX-License-Identifier: UNLICENSED
pragma solidity >=0.8.25 <0.9.0;

import { stdJson } from "forge-std/StdJson.sol";
import { Script } from "forge-std/Script.sol";
import { TestERC20 } from "./TestERC20.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";
import { ICS20Lib } from "./ICS20Lib.sol";
import { FastTransferGateway } from "./FastTransferGateway.sol";
import { ERC1967Proxy } from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

contract MockE2ETestDeploy is Script {
    using stdJson for string;

    string public constant E2E_FAUCET = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266";
    address public constant MOCK_MAILBOX = 0x0000000000000000000000000000000000000001;
    address public constant MOCK_PERMIT2 = 0x0000000000000000000000000000000000000002;
    address public constant MOCK_ISM = 0x0000000000000000000000000000000000000003;

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
            MOCK_MAILBOX,       // Mailbox
            MOCK_ISM,          // InterchainSecurityModule
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

        string memory json = "json";
        json.serialize("erc20", Strings.toHexString(address(erc20)));
        string memory finalJson = json.serialize("fast_transfer_gateway", Strings.toHexString(address(gatewayProxy)));

        return finalJson;
    }
}