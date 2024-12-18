pragma solidity ^0.8.13;

import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import {OwnableUpgradeable} from "@openzeppelin-contracts-upgradeable/contracts/access/OwnableUpgradeable.sol";
import {Initializable} from "@openzeppelin-contracts-upgradeable/contracts/proxy/utils/Initializable.sol";
import {UUPSUpgradeable} from "@openzeppelin/contracts/proxy/utils/UUPSUpgradeable.sol";
import {ReentrancyGuardUpgradeable} from "@openzeppelin-contracts-upgradeable/contracts/security/ReentrancyGuardUpgradeable.sol";

import {TypeCasts} from "./libraries/TypeCasts.sol";
import {OrderEncoder} from "./libraries/OrderEncoder.sol";

import {IPermit2} from "./interfaces/IPermit2.sol";
import {IMailbox} from "./interfaces/hyperlane/IMailbox.sol";

// Structure that contains the order details required to settle or refund an order

struct SettlementDetails {
    // The sender of the order
    bytes32 sender;
    // The nonce of the order
    uint256 nonce;
    // The destination domain of the order
    uint32 destinationDomain;
    // The amount of the order
    uint256 amount;
}

// Structure that contains the full details of an order

struct FastTransferOrder {
    // The sender of the order on the source domain
    bytes32 sender;
    // The recipient of the order on the destination domain
    bytes32 recipient;
    // The amount of tokens the user is sending on the source domain
    uint256 amountIn;
    // The amount of tokens the user is receiving on the destination domain
    uint256 amountOut;
    // Nonce of the order
    uint256 nonce;
    // Source domain of the order
    uint32 sourceDomain;
    // Destination domain of the order
    uint32 destinationDomain;
    // Deadline that the order must be filled on the destination domain by
    uint256 timeoutTimestamp;
    // Optional calldata passed on to the recipient on the destination domain when the order is filled
    bytes data;
}

// Structure that contains the order fill details
struct OrderFill {
    // The ID of the order
    bytes32 orderID;
    // The address that filled the order
    address filler;
    // The source domain of the order
    uint32 sourceDomain;
}

enum Command {
    SETTLE_ORDERS,
    REFUND_ORDERS
}

enum OrderStatus {
    UNFILLED,
    FILLED,
    REFUNDED
}

contract FastTransferGateway is Initializable, UUPSUpgradeable, OwnableUpgradeable, ReentrancyGuardUpgradeable {
    IPermit2 public PERMIT2;
    address public token;
    address public mailbox;

    // TODO: make this immutable after discussing with the team
    uint32 public localDomain;

    mapping(uint32 => bytes32) public remoteDomains;

    uint32 public nonce;

    mapping(bytes32 => SettlementDetails) public settlementDetails;
    mapping(bytes32 => OrderStatus) public orderStatuses;

    mapping(bytes32 => OrderFill) public orderFills;

    constructor() {
        _disableInitializers();
    }

    function initialize(uint32 _localDomain, address _owner, address _token, address _mailbox, address _permit2)
        external
        initializer
    {
        __Ownable_init();
        _transferOwnership(_owner);

        token = _token;
        mailbox = _mailbox;
        localDomain = _localDomain;
        PERMIT2 = IPermit2(_permit2);
        nonce = 1;
    }

    /// @dev Emitted when an order is submitted
    /// @param orderID The ID of the order
    /// @param order The details of the order
    event OrderSubmitted(bytes32 indexed orderID, bytes order);

    /// @dev Emitted when an order is settled
    /// @param orderID The ID of the order
    event OrderSettled(bytes32 indexed orderID);

    /// @dev Emitted when an order is requested to be settled but is already settled
    /// @param orderID The ID of the order
    event OrderAlreadySettled(bytes32 indexed orderID);

    /// @dev Emitted when an order is refunded
    /// @param orderID The ID of the order
    event OrderRefunded(bytes32 indexed orderID);

    /// @dev Throws if the sender is not the configured Hyperlane mailbox
    modifier onlyMailbox() {
        require(msg.sender == address(mailbox), "FastTransferGateway: sender not mailbox");
        _;
    }

    /// @dev Sets the remote domain contract for a domain
    /// @param domain The domain to set the remote domain contract for
    /// @param remoteContract The remote domain contract
    function setRemoteDomain(uint32 domain, bytes32 remoteContract) public onlyOwner {
        remoteDomains[domain] = remoteContract;
    }

    /// @dev Submits a fast transfer order to the gateway
    /// @param sender The sender of the order used in the case of a timeout
    /// @param recipient The recipient of the order on the destination domain
    /// @param amountIn The amount of tokens the user is sending on the source domain
    /// @param amountOut The amount of tokens the user is receiving on the destination domain
    /// @param destinationDomain The destination domain of the order
    /// @param timeoutTimestamp The deadline that the order must be filled on the destination domain by
    /// @param data Optional calldata passed on to the recipient on the destination domain when the order is filled
    function submitOrder(
        bytes32 sender,
        bytes32 recipient,
        uint256 amountIn,
        uint256 amountOut,
        uint32 destinationDomain,
        uint256 timeoutTimestamp,
        bytes calldata data
    ) public returns (bytes32) {
        FastTransferOrder memory order = FastTransferOrder(
            sender, recipient, amountIn, amountOut, nonce, localDomain, destinationDomain, timeoutTimestamp, data
        );

        bytes32 orderID = _orderID(order);

        // checks
        require(remoteDomains[destinationDomain] != bytes32(0), "FastTransferGateway: destination domain not found");

        // effects
        nonce += 1;
        settlementDetails[orderID] =
            SettlementDetails(order.sender, order.nonce, order.destinationDomain, order.amountIn);

        emit OrderSubmitted(orderID, OrderEncoder.encode(order));

        // interactions
        SafeERC20.safeTransferFrom(IERC20(token), msg.sender, address(this), order.amountIn);

        return orderID;
    }

    /// @dev Submits a fast transfer order to the gateway with a permit signature
    /// @param sender The sender of the order used in the case of a timeout
    /// @param recipient The recipient of the order on the destination domain
    /// @param amountIn The amount of tokens the user is sending on the source domain
    /// @param amountOut The amount of tokens the user is receiving on the destination domain
    /// @param destinationDomain The destination domain of the order
    /// @param timeoutTimestamp The deadline that the order must be filled on the destination domain by
    /// @param permitDeadline The deadline that the permit is valid for
    /// @param data Optional calldata passed on to the recipient on the destination domain when the order is filled
    /// @param signature The signature of the permit
    function submitOrderWithPermit(
        bytes32 sender,
        bytes32 recipient,
        uint256 amountIn,
        uint256 amountOut,
        uint32 destinationDomain,
        uint256 timeoutTimestamp,
        uint256 permitDeadline,
        bytes calldata data,
        bytes calldata signature
    ) public returns (bytes32) {
        FastTransferOrder memory order = FastTransferOrder(
            sender, recipient, amountIn, amountOut, nonce, localDomain, destinationDomain, timeoutTimestamp, data
        );

        bytes32 orderID = _orderID(order);

        // checks
        require(remoteDomains[destinationDomain] != bytes32(0), "FastTransferGateway: destination domain not found");

        // effects
        nonce += 1;
        settlementDetails[orderID] =
            SettlementDetails(order.sender, order.nonce, order.destinationDomain, order.amountIn);

        emit OrderSubmitted(orderID, OrderEncoder.encode(order));

        // interactions
        _permitTransferFrom(order.amountIn, permitDeadline, order.nonce, signature);

        return orderID;
    }

    /// @dev Fills an order
    /// @param order The details of the order to fill
    function fillOrder(address filler, FastTransferOrder memory order) public nonReentrant {
        require(order.timeoutTimestamp > block.timestamp, "FastTransferGateway: order expired");
        require(remoteDomains[order.sourceDomain] != bytes32(0), "FastTransferGateway: source domain not found");
        require(order.destinationDomain == localDomain, "FastTransferGateway: incorrect destination domain for order");

        bytes32 orderID = _orderID(order);

        require(orderStatuses[orderID] == OrderStatus.UNFILLED, "FastTransferGateway: order already filled");

        address recipient = TypeCasts.bytes32ToAddress(order.recipient);

        orderStatuses[orderID] = OrderStatus.FILLED;
        orderFills[orderID] = OrderFill(orderID, filler, order.sourceDomain);

        if (order.data.length > 0) {
            SafeERC20.safeTransferFrom(IERC20(token), msg.sender, address(this), order.amountOut);
            IERC20(token).approve(address(recipient), order.amountOut);
            (bool success,) = address(recipient).call(order.data);
            if (!success) {
                assembly {
                    returndatacopy(0, 0, returndatasize())
                    revert(0, returndatasize())
                }
            }
        } else {
            SafeERC20.safeTransferFrom(IERC20(token), msg.sender, recipient, order.amountOut);
        }
    }

    /// @dev Initiates a settlement for a batch of orders
    /// @param repaymentAddress The address to repay the orders to
    /// @param orderIDs The IDs of the orders to settle
    function initiateSettlement(bytes32 repaymentAddress, bytes memory orderIDs) public payable {
        uint32 sourceDomain;
        for (uint256 pos = 0; pos < orderIDs.length; pos += 32) {
            bytes32 orderID;
            assembly {
                orderID := mload(add(orderIDs, add(0x20, pos)))
            }

            OrderFill memory orderFill = _fillByOrderID(orderID);
            require(orderFill.filler == msg.sender, "FastTransferGateway: Unauthorized");
            if (pos != 0) {
                require(orderFill.sourceDomain == sourceDomain, "FastTransferGateway: Source domains must match");
            }

            sourceDomain = orderFill.sourceDomain;
        }

        bytes32 remoteContract = remoteDomains[sourceDomain];
        require(remoteContract != bytes32(0), "FastTransferGateway: unknown source domain");

        bytes memory hyperlaneMessage = abi.encodePacked(uint8(Command.SETTLE_ORDERS), repaymentAddress, orderIDs);

        IMailbox(mailbox).dispatch{value: msg.value}(sourceDomain, remoteContract, hyperlaneMessage);
    }

    function initiateTimeout(FastTransferOrder[] memory orders) public payable {
        bytes memory orderIDs;
        uint32 sourceDomain;
        for (uint256 i = 0; i < orders.length; i++) {
            FastTransferOrder memory order = orders[i];
            bytes32 orderID = _orderID(order);
            OrderStatus status = orderStatuses[orderID];

            require(order.timeoutTimestamp < block.timestamp, "FastTransferGateway: order not timed out");
            require(status == OrderStatus.UNFILLED, "FastTransferGateway: order filled");
            if (i != 0) {
                require(order.sourceDomain == sourceDomain, "FastTransferGateway: Source domains must match");
            }

            orderIDs = bytes.concat(orderIDs, orderID);
            sourceDomain = order.sourceDomain;
        }

        bytes32 remoteContract = remoteDomains[sourceDomain];
        require(remoteContract != bytes32(0), "FastTransferGateway: unknown source domain");

        bytes memory hyperlaneMessage = abi.encodePacked(uint8(Command.REFUND_ORDERS), orderIDs);

        IMailbox(mailbox).dispatch{value: msg.value}(sourceDomain, remoteContract, hyperlaneMessage);
    }

    function quoteInitiateSettlement(uint32 sourceDomain, bytes32 repaymentAddress, bytes memory orderIDs)
        public
        view
        returns (uint256)
    {
        bytes32 remoteContract = remoteDomains[sourceDomain];
        require(remoteContract != bytes32(0), "FastTransferGateway: unknown source domain");

        bytes memory hyperlaneMessage = abi.encodePacked(uint8(Command.SETTLE_ORDERS), repaymentAddress, orderIDs);

        return IMailbox(mailbox).quoteDispatch(sourceDomain, remoteContract, hyperlaneMessage);
    }

    function quoteInitiateTimeout(uint32 sourceDomain, FastTransferOrder[] memory orders)
        public
        view
        returns (uint256)
    {
        bytes32 remoteContract = remoteDomains[sourceDomain];
        require(remoteContract != bytes32(0), "FastTransferGateway: unknown source domain");

        bytes memory orderIDs;
        for (uint256 i = 0; i < orders.length; i++) {
            orderIDs = bytes.concat(orderIDs, _orderID(orders[i]));
        }

        bytes memory hyperlaneMessage = abi.encodePacked(uint8(Command.REFUND_ORDERS), orderIDs);

        return IMailbox(mailbox).quoteDispatch(sourceDomain, remoteContract, hyperlaneMessage);
    }

    /// @dev Handles a message from a remote domain
    /// @dev This function can only be called by the hyperlane mailbox
    /// @param _origin The origin domain of the message
    /// @param _sender The sender of the message (must be the configured remote domain contract for the domain the message originated from)
    /// @param _message The message
    function handle(uint32 _origin, bytes32 _sender, bytes calldata _message) external payable onlyMailbox {
        bytes32 remoteContract = remoteDomains[_origin];

        require(remoteContract != bytes32(0), "FastTransferGateway: origin domain not found");
        require(_sender == remoteContract, "FastTransferGateway: invalid sender");

        Command command = Command(uint8(_message[0]));

        if (command == Command.SETTLE_ORDERS) {
            bytes calldata payload = _message[1:];

            bytes32 repaymentAddressBytes = bytes32(payload[:32]);
            bytes memory orderIDs = payload[32:];

            address repaymentAddress = TypeCasts.bytes32ToAddress(repaymentAddressBytes);

            _settleOrder(repaymentAddress, orderIDs, _origin);
        } else if (command == Command.REFUND_ORDERS) {
            bytes calldata payload = _message[1:];

            _refundOrders(payload, _origin);
        }
    }

    function _settleOrder(address repaymentAddress, bytes memory orderIDs, uint32 domain) internal {
        uint256 amountToRepay = 0;

        // checks
        for (uint256 pos = 0; pos < orderIDs.length; pos += 32) {
            bytes32 orderID;
            assembly {
                orderID := mload(add(orderIDs, add(0x20, pos)))
            }

            SettlementDetails memory orderSettlementDetails = settlementDetails[orderID];

            if (orderStatuses[orderID] != OrderStatus.UNFILLED) {
                continue;
            }

            require(orderSettlementDetails.nonce > 0, "FastTransferGateway: order not found");
            require(
                orderSettlementDetails.destinationDomain == domain,
                "FastTransferGateway: incorrect domain for settlement"
            );

            amountToRepay += orderSettlementDetails.amount;
        }

        // effects
        for (uint256 pos = 0; pos < orderIDs.length; pos += 32) {
            bytes32 orderID;
            assembly {
                orderID := mload(add(orderIDs, add(0x20, pos)))
            }

            if (orderStatuses[orderID] != OrderStatus.UNFILLED) {
                emit OrderAlreadySettled(orderID);
                continue;
            }

            orderStatuses[orderID] = OrderStatus.FILLED;
            emit OrderSettled(orderID);
        }

        // interactions
        SafeERC20.safeTransfer(IERC20(token), repaymentAddress, amountToRepay);
    }

    function _refundOrders(bytes memory orderIDs, uint32 domain) internal {
        for (uint256 pos = 0; pos < orderIDs.length; pos += 32) {
            bytes32 orderID;
            assembly {
                orderID := mload(add(orderIDs, add(0x20, pos)))
            }

            _refundOrder(orderID, domain);
        }
    }

    function _refundOrder(bytes32 orderID, uint32 domain) internal {
        SettlementDetails memory orderSettlementDetails = settlementDetails[orderID];

        require(orderSettlementDetails.nonce > 0, "FastTransferGateway: order not found");
        require(
            orderSettlementDetails.destinationDomain == domain, "FastTransferGateway: incorrect domain for settlement"
        );

        if (orderStatuses[orderID] != OrderStatus.UNFILLED) {
            return;
        }

        orderStatuses[orderID] = OrderStatus.REFUNDED;

        SafeERC20.safeTransfer(
            IERC20(token), TypeCasts.bytes32ToAddress(orderSettlementDetails.sender), orderSettlementDetails.amount
        );

        emit OrderRefunded(orderID);
    }

    function _permitTransferFrom(uint256 amount, uint256 deadline, uint256 orderNonce, bytes calldata signature)
        internal
    {
        PERMIT2.permitTransferFrom(
            // The permit message. Spender will be inferred as the caller (us).
            IPermit2.PermitTransferFrom({
                permitted: IPermit2.TokenPermissions({token: IERC20(token), amount: amount}),
                nonce: orderNonce,
                deadline: deadline
            }),
            // The transfer recipient and amount.
            IPermit2.SignatureTransferDetails({to: address(this), requestedAmount: amount}),
            // The owner of the tokens, which must also be
            // the signer of the message, otherwise this call
            // will fail.
            msg.sender,
            // The packed signature that was the result of signing
            // the EIP712 hash of `permit`.
            signature
        );
    }

    function _orderID(FastTransferOrder memory order) internal pure returns (bytes32) {
        return OrderEncoder.id(order);
    }

    function _fillByOrderID(bytes32 orderID) internal view returns (OrderFill memory) {
        OrderFill memory orderFill = orderFills[orderID];
        require(orderFill.filler != address(0), "FastTransferGateway: order not filled");

        return orderFill;
    }

    function _authorizeUpgrade(address newImplementation) internal override onlyOwner {}
}
