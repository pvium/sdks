from eth_abi import encode
from eth_account import Account
from eth_account.messages import encode_defunct
from eth_utils import keccak, to_hex

from pvium_sdk import (
    PVIUM_SIGNATURE_DOMAIN,
    createSignerFromPrivateKey,
    hashCreateClaimRequest,
    hashDisputeRequest,
    hashFinalizeClaimRequest,
    hashCreateProjectRequest,
    hashRelayedCallRequest,
    hashResolveDisputeRequest,
    signCreateClaimRequest,
    signDisputeRequest,
    signFinalizeClaimRequest,
    signCreateProjectRequest,
    signRelayedCallRequest,
    signResolveDisputeRequest,
)


TEST_PRIVATE_KEY = "0x59c6995e998f97a5a0044976f0d7f3f6f8f53f6a2046baf4f01cb4f1f6bcb58f"
TEST_ADDRESS = Account.from_key(TEST_PRIVATE_KEY).address
CHAIN_ID = 84532


def test_create_signer_from_private_key():
    signer = createSignerFromPrivateKey(TEST_PRIVATE_KEY)
    assert signer.address == TEST_ADDRESS

    message_hash = to_hex(keccak(b"hello-pvium"))
    signature = signer.sign_message(encode_defunct(hexstr=message_hash)).signature.hex()
    recovered = Account.recover_message(encode_defunct(hexstr=message_hash), signature=signature)
    assert recovered == TEST_ADDRESS


def test_sign_create_project_matches_manual_hash_encoding():
    payload = {
        "app": "test-app",
        "projectId": "project-001",
        "metadata": "ipfs://QmTest",
        "tokenAddress": "0x0000000000000000000000000000000000000001",
        "refundAddress": "0x0000000000000000000000000000000000000002",
        "appFeeAddress": "0x0000000000000000000000000000000000000003",
        "appAdminAddress": "0x0000000000000000000000000000000000000004",
        "appFeeBps": 200,
        "disputeWindowSeconds": 259200,
        "lockDuration": 7776000,
        "minimumBalancePerVendor": 100000000,
    }
    options = {"pviumFeeBps": 100, "chainId": CHAIN_ID}

    expected_hash = to_hex(
        keccak(
            encode(
                [
                    "bytes32",
                    "string",
                    "string",
                    "string",
                    "address",
                    "address",
                    "address",
                    "address",
                    "uint256",
                    "uint256",
                    "uint256",
                    "uint256",
                    "uint256",
                    "uint256",
                ],
                [
                    bytes.fromhex(PVIUM_SIGNATURE_DOMAIN[2:]),
                    payload["app"],
                    payload["projectId"],
                    payload["metadata"],
                    payload["tokenAddress"],
                    payload["refundAddress"],
                    payload["appFeeAddress"],
                    payload["appAdminAddress"],
                    payload["appFeeBps"],
                    payload["disputeWindowSeconds"],
                    payload["lockDuration"],
                    payload["minimumBalancePerVendor"],
                    options["pviumFeeBps"],
                    options["chainId"],
                ],
            )
        )
    )

    helper_hash = hashCreateProjectRequest(payload, options)
    assert helper_hash == expected_hash

    signature = signCreateProjectRequest(payload, TEST_PRIVATE_KEY, options)
    recovered = Account.recover_message(encode_defunct(hexstr=helper_hash), signature=signature)
    assert recovered == TEST_ADDRESS


def test_sign_create_claim_request_matches_manual_hash_encoding():
    payload = {
        "app": "test-app",
        "projectId": "project-001",
        "claimId": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        "receiver": "0x0000000000000000000000000000000000000005",
        "amount": 100000000,
        "claimableAfter": 1700000000,
        "claimDeadline": 0,
        "nonce": 1,
    }

    expected_hash = to_hex(
        keccak(
            encode(
                ["string", "string", "bytes32", "address", "uint256", "uint256", "uint256", "uint256"],
                [
                    payload["app"],
                    payload["projectId"],
                    bytes.fromhex(payload["claimId"][2:]),
                    payload["receiver"],
                    payload["amount"],
                    payload["claimableAfter"],
                    payload["claimDeadline"],
                    payload["nonce"],
                ],
            )
        )
    )

    helper_hash = hashCreateClaimRequest(payload)
    assert helper_hash == expected_hash

    signature = signCreateClaimRequest(payload, TEST_PRIVATE_KEY)
    recovered = Account.recover_message(encode_defunct(hexstr=helper_hash), signature=signature)
    assert recovered == TEST_ADDRESS


def test_sign_finalize_claim_request_hashes_packed_batch_payload_like_contract_tests():
    claims = [
        {"app": "test-app", "projectId": "usdc-project", "claimId": "0x" + "bb" * 32},
        {"app": "test-app", "projectId": "usdt-project", "claimId": "0x" + "cc" * 32},
    ]

    data_packed = b""
    for claim in claims:
        data_packed += claim["app"].encode("utf-8")
        data_packed += claim["projectId"].encode("utf-8")
        data_packed += bytes.fromhex(claim["claimId"][2:])
    expected_hash = to_hex(keccak(data_packed + CHAIN_ID.to_bytes(32, byteorder="big")))

    helper_hash = hashFinalizeClaimRequest(claims, CHAIN_ID)
    assert helper_hash == expected_hash

    signature = signFinalizeClaimRequest(claims, TEST_PRIVATE_KEY, CHAIN_ID)
    recovered = Account.recover_message(encode_defunct(hexstr=helper_hash), signature=signature)
    assert recovered == TEST_ADDRESS


def test_relayed_dispute_resolve_helpers_match_manual_encoding_and_accept_signer_instance():
    signer = createSignerFromPrivateKey(TEST_PRIVATE_KEY)
    relayed_payload = {
        "appId": "test-app",
        "projectId": "project-001",
        "payload": to_hex(encode(["string", "address[]"], ["addVendors", []])),
        "nonce": 2,
        "chainId": CHAIN_ID,
    }

    expected_relayed_hash = to_hex(
        keccak(
            encode(
                ["string", "string", "bytes", "uint256", "uint256"],
                [
                    relayed_payload["appId"],
                    relayed_payload["projectId"],
                    bytes.fromhex(relayed_payload["payload"][2:]),
                    relayed_payload["nonce"],
                    relayed_payload["chainId"],
                ],
            )
        )
    )
    relayed_hash = hashRelayedCallRequest(relayed_payload)
    assert relayed_hash == expected_relayed_hash
    relayed_sig = signRelayedCallRequest(relayed_payload, signer)
    assert Account.recover_message(encode_defunct(hexstr=relayed_hash), signature=relayed_sig) == TEST_ADDRESS

    claim_id = "0x" + "dd" * 32
    expected_dispute_hash = to_hex(keccak(encode(["bytes32", "uint256"], [bytes.fromhex(claim_id[2:]), CHAIN_ID])))
    dispute_hash = hashDisputeRequest(claim_id, CHAIN_ID)
    assert dispute_hash == expected_dispute_hash
    dispute_sig = signDisputeRequest(claim_id, signer, CHAIN_ID)
    assert Account.recover_message(encode_defunct(hexstr=dispute_hash), signature=dispute_sig) == TEST_ADDRESS

    resolve_payload = {"claimId": claim_id, "approved": True, "chainId": CHAIN_ID}
    expected_resolve_hash = to_hex(keccak(encode(["bytes32", "bool", "uint256"], [bytes.fromhex(claim_id[2:]), True, CHAIN_ID])))
    resolve_hash = hashResolveDisputeRequest(resolve_payload)
    assert resolve_hash == expected_resolve_hash
    resolve_sig = signResolveDisputeRequest(resolve_payload, signer)
    assert Account.recover_message(encode_defunct(hexstr=resolve_hash), signature=resolve_sig) == TEST_ADDRESS
