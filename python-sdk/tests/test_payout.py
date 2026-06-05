import json
import re
from pathlib import Path
from urllib.parse import parse_qs, urlparse

from eth_abi import encode
from eth_abi.packed import encode_packed
from eth_utils import keccak, to_hex

from pvium_sdk import (
    PayoutCurrency,
    PayoutFinalization,
    PayoutIntent,
    PviumSdk,
    PviumSdkConfig,
    computeEscrowFundingDigest,
    computeEscrowScheduledFundingDigest,
    computeScheduledPayoutHash,
    createPayoutNonce,
    generateInstantPayoutHash,
)


def load_parity_fixture():
    path = Path(__file__).resolve().parents[2] / "parity-fixtures" / "scheduled-payout-finalize.json"
    return json.loads(path.read_text())


def make_fetch(response_bodies, requests):
    queue = list(response_bodies)

    def _fetch(method, url, headers, payload, timeout):
        requests.append({"method": method, "url": url, "headers": headers, "payload": payload})
        body = queue.pop(0)
        return body.get("meta", {}).get("statusCode", 200), {"content-type": "application/json"}, json.dumps(body)

    return _fetch


def test_create_payout_nonce_hex_format():
    nonce = createPayoutNonce()
    assert re.match(r"^0x[0-9a-f]{32}$", nonce)


def test_sdk_init_exposes_payout_service():
    sdk = PviumSdk.init(PviumSdkConfig(baseUrl="https://api.example.test/v1", apiKey="app_key"))

    assert callable(sdk.payout.create)
    assert callable(sdk.payout.finalize)
    assert callable(sdk.payout.addPayments)
    assert callable(sdk.payout.addRecipients)
    assert callable(sdk.payout.resolveRecipients)
    assert callable(sdk.payout.removePayments)
    assert callable(sdk.payout.deletePayment)
    assert callable(sdk.payout.updatePayment)
    assert callable(sdk.payout.editPayment)
    assert callable(sdk.payout.listPayments)
    assert callable(sdk.payout.listInvites)
    assert callable(sdk.payout.revokeInvite)
    assert callable(sdk.payout.revokeInviteRoot)
    assert callable(sdk.payout.delete)


def test_create_returns_payout_intent_with_proxy_methods():
    private_key = "0x59c6995e998f97a5a0044976f0d7f3f6f8f53f6a2046baf4f01cb4f1f6bcb58f"
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            clientId="app_test",
            fetchFn=make_fetch(
                [
                    {
                        "meta": {"statusCode": 201, "success": True},
                        "data": {
                            "id": "batch_1",
                            "chain": "base",
                            "paymentType": "Instant",
                            "nonce": "0x11111111111111111111111111111111",
                            "complianceMode": "Open",
                            "payments": [
                                {
                                    "receiver": "0x0000000000000000000000000000000000000001",
                                    "amount": 25,
                                    "token": "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
                                    "decimals": 6,
                                }
                            ],
                        },
                    },
                    {
                        "meta": {"statusCode": 200, "success": True},
                        "data": {
                            "id": "batch_1",
                            "chain": "base",
                            "paymentType": "Instant",
                            "nonce": "0x11111111111111111111111111111111",
                            "batchDataHash": "0x" + "22" * 32,
                        },
                    },
                ],
                requests,
            ),
        )
    )

    payout_intent = sdk.payout.create(
        {
            "type": "Instant",
            "chain": "base",
            "name": "Creator payroll",
            "payments": [
                {
                    "receiver": "0x0000000000000000000000000000000000000001",
                    "amount": 25,
                    "token": "usdc",
                }
            ],
        }
    )

    assert isinstance(payout_intent, PayoutIntent)
    assert payout_intent.id == "batch_1"
    assert payout_intent["data"]["id"] == "batch_1"

    finalized = payout_intent.finalize(private_key, {"timestamp": 1777487451})

    assert isinstance(finalized, PayoutFinalization)
    assert finalized.payout.id == "batch_1"
    assert finalized["data"]["payout"]["id"] == "batch_1"
    assert requests[1]["method"] == "PATCH"


def test_generate_instant_payout_hash_matches_manual_encoding():
    nonce = "0x1234abcd"
    payments = [
        {
            "receiver": "0x0000000000000000000000000000000000000001",
            "amount": "12.5",
            "token": "0x0000000000000000000000000000000000000002",
            "decimals": 6,
            "memo": "first",
        },
        {
            "receiver": "0x0000000000000000000000000000000000000003",
            "amount": 1,
            "token": "0x0000000000000000000000000000000000000002",
            "decimals": 6,
        },
    ]

    expected = to_hex(
        keccak(
            encode(
                ["bytes", "(address,uint256,address,string)[]"],
                [
                    bytes.fromhex(nonce[2:]),
                    [
                        (
                            payments[0]["receiver"],
                            12500000,
                            payments[0]["token"],
                            "first",
                        ),
                        (
                            payments[1]["receiver"],
                            1000000,
                            payments[1]["token"],
                            "",
                        ),
                    ],
                ],
            )
        )
    )

    assert generateInstantPayoutHash(payments, nonce) == expected


def test_compute_scheduled_payout_hash_matches_formula():
    params = {
        "payoutId": "120bdabb-5790-415c-ae75-c2fca1cc5232",
        "fundingToken": "0x0000000000000000000000000000000000000002",
        "gracePeriod": 86400,
        "disapprovalDeadline": 3600,
        "timestamp": 1777487451,
        "chainId": 8453,
    }

    payout_id_bytes32 = bytes.fromhex(params["payoutId"].replace("-", "").ljust(64, "0"))
    expected = to_hex(
        keccak(
            encode(
                ["bytes32", "address", "uint256", "uint256", "uint256", "uint256"],
                [
                    payout_id_bytes32,
                    params["fundingToken"],
                    params["gracePeriod"],
                    params["disapprovalDeadline"],
                    params["timestamp"],
                    params["chainId"],
                ],
            )
        )
    )

    assert computeScheduledPayoutHash(params) == expected


def test_add_recipients_posts_to_open_payees_endpoint():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch(
                [{"meta": {"statusCode": 201, "success": True}, "data": {"added": [], "errors": []}}],
                requests,
            ),
        )
    )

    sdk.payout.addRecipients(
        "batch 1",
        [
            {
                "identityType": "github",
                "identityValue": "@feminefa",
                "defaultPayoutAmount": 25,
                "memo": "github payout",
            }
        ],
    )

    assert requests[0]["url"] == "http://localhost:4005/v1/batch-payments/batch%201/open-payees"
    assert requests[0]["method"] == "POST"


def test_resolve_recipients_posts_recipient_identities_to_resolver_endpoint():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch(
                [{"meta": {"statusCode": 200, "success": True}, "data": {"resolved": [], "errors": []}}],
                requests,
            ),
        )
    )

    sdk.payout.resolveRecipients("batch_1", [{"identityType": "email", "identityValue": "payee@example.com"}])

    assert requests[0]["url"] == "http://localhost:4005/v1/batch-payments/batch_1/resolve-recipients"
    assert requests[0]["method"] == "POST"
    assert json.loads(requests[0]["payload"]) == {"recipients": [{"identityType": "email", "identityValue": "payee@example.com"}]}


def test_remove_payments_deletes_payment_ids_from_payout():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch([{"meta": {"statusCode": 200, "success": True}}], requests),
        )
    )

    sdk.payout.removePayments("batch_1", ["1", 2])

    assert requests[0]["url"] == "http://localhost:4005/v1/batch-payments/batch_1/payments"
    assert requests[0]["method"] == "DELETE"
    assert json.loads(requests[0]["payload"]) == {"paymentIds": [1, 2]}


def test_list_payments_requests_paginated_payout_payments():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch(
                [{"meta": {"statusCode": 200, "success": True, "pagination": {"totalCount": 251}}, "data": []}],
                requests,
            ),
        )
    )

    response = sdk.payout.listPayments("batch_1", {"page": 2, "perPage": 50})

    url = urlparse(requests[0]["url"])
    assert url.path == "/v1/batch-payments/batch_1/payments"
    assert parse_qs(url.query) == {"page": ["2"], "perPage": ["50"]}
    assert requests[0]["method"] == "GET"
    assert response["meta"]["pagination"]["totalCount"] == 251


def test_payout_intent_proxies_payment_management_routes():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch(
                [
                    {"meta": {"statusCode": 200, "success": True}, "data": [{"id": 77}]},
                    {"meta": {"statusCode": 200, "success": True}, "data": {"id": 77, "memo": "Updated"}},
                    {"meta": {"statusCode": 200, "success": True}, "data": {}},
                ],
                requests,
            ),
        )
    )
    payout_intent = PayoutIntent(sdk.payout, {"statusCode": 200, "success": True}, {"id": "batch_1"})

    payout_intent.listPayments({"page": 1, "perPage": 25})
    payout_intent.editPayment(77, {"memo": "Updated"})
    payout_intent.deletePayment(77)

    assert urlparse(requests[0]["url"]).path == "/v1/batch-payments/batch_1/payments"
    assert parse_qs(urlparse(requests[0]["url"]).query) == {"page": ["1"], "perPage": ["25"]}
    assert requests[1]["method"] == "PATCH"
    assert requests[1]["url"] == "http://localhost:4005/v1/batch-payments/batch_1/payments/77"
    assert json.loads(requests[1]["payload"]) == {"memo": "Updated"}
    assert requests[2]["method"] == "DELETE"
    assert json.loads(requests[2]["payload"]) == {"paymentIds": [77]}


def test_create_maps_direct_scheduled_payout_currency_and_date_into_metadata():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch(
                [{"meta": {"statusCode": 201, "success": True}, "data": {"id": "batch_1", "paymentType": "Scheduled"}}],
                requests,
            ),
        )
    )

    sdk.payout.create(
        {
            "type": "Scheduled",
            "chain": "base",
            "name": "March creator payouts",
            "payoutCurrency": PayoutCurrency.USDC,
            "scheduleDate": 1777488000,
            "metadata": {"campaign": "march"},
            "payments": [{"receiver": "0x0000000000000000000000000000000000000001", "amount": "25"}],
        }
    )

    payload = json.loads(requests[0]["payload"])
    assert payload["metadata"]["payoutCurrency"] == "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
    assert payload["metadata"]["scheduledDate"] == 1777488000
    assert payload["metadata"]["campaign"] == "march"
    assert payload["payments"][0]["token"] == "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
    assert payload["payments"][0]["decimals"] == 6


def test_create_maps_direct_currency_from_supported_network_config():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch(
                [{"meta": {"statusCode": 201, "success": True}, "data": {"id": "batch_1", "paymentType": "Scheduled"}}],
                requests,
            ),
        )
    )

    sdk.payout.create(
        {
            "type": "Scheduled",
            "chain": "solana-testnet",
            "payoutCurrency": "usdt",
            "payments": [{"receiver": "recipient", "amount": "25"}],
        }
    )

    payload = json.loads(requests[0]["payload"])
    assert payload["metadata"]["payoutCurrency"] == "SPFPKg9zeE7ReqW3j9QU6p7XhPP8JDU5Dx4fgrTwVyF"
    assert payload["payments"][0]["token"] == "SPFPKg9zeE7ReqW3j9QU6p7XhPP8JDU5Dx4fgrTwVyF"
    assert payload["payments"][0]["decimals"] == 6


def test_create_resolves_payment_token_symbols_into_configured_addresses():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch(
                [{"meta": {"statusCode": 201, "success": True}, "data": {"id": "batch_1", "paymentType": "Scheduled"}}],
                requests,
            ),
        )
    )

    sdk.payout.create(
        {
            "type": "Scheduled",
            "chain": "base",
            "payments": [
                {"receiver": "0x0000000000000000000000000000000000000001", "amount": "25", "token": "usdt"},
                {"receiver": "0x0000000000000000000000000000000000000002", "amount": "10", "tokenSymbol": "usdc"},
            ],
        }
    )

    payload = json.loads(requests[0]["payload"])
    assert payload["payments"][0]["token"] == "0xfde4C96c8593536E31F229EA8f37b2ADa2699bb2"
    assert payload["payments"][0]["decimals"] == 6
    assert payload["payments"][1]["token"] == "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
    assert payload["payments"][1]["decimals"] == 6
    assert "tokenSymbol" not in payload["payments"][1]


def test_create_rejects_payment_token_mismatches_when_payout_currency_is_provided():
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch(
                [{"meta": {"statusCode": 201, "success": True}, "data": {"id": "batch_1", "paymentType": "Scheduled"}}],
                [],
            ),
        )
    )

    try:
        sdk.payout.create(
            {
                "type": "Scheduled",
                "chain": "base",
                "payoutCurrency": "usdc",
                "payments": [{"receiver": "0x0000000000000000000000000000000000000001", "amount": "25", "token": "usdt"}],
            }
        )
    except RuntimeError as err:
        assert "Payment token must match payoutCurrency" in str(err)
    else:
        raise AssertionError("expected mismatch error")


def test_create_validates_explicit_payment_token_addresses_against_supported_config():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch(
                [{"meta": {"statusCode": 201, "success": True}, "data": {"id": "batch_1", "paymentType": "Scheduled"}}],
                requests,
            ),
        )
    )

    sdk.payout.create(
        {
            "type": "Scheduled",
            "chain": "base",
            "payments": [
                {
                    "receiver": "0x0000000000000000000000000000000000000001",
                    "amount": "25",
                    "token": "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
                }
            ],
        }
    )

    payload = json.loads(requests[0]["payload"])
    assert payload["payments"][0]["token"] == "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
    assert payload["payments"][0]["decimals"] == 6

    try:
        sdk.payout.create(
            {
                "type": "Scheduled",
                "chain": "base",
                "payments": [
                    {
                        "receiver": "0x0000000000000000000000000000000000000001",
                        "amount": "25",
                        "token": "0x0000000000000000000000000000000000000009",
                    }
                ],
            }
        )
    except RuntimeError as err:
        assert "is not supported on chain base" in str(err)
    else:
        raise AssertionError("expected unsupported token error")


def test_create_injects_milestone_commitment_metadata():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch(
                [{"meta": {"statusCode": 201, "success": True}, "data": {"id": "batch_1", "paymentType": "Scheduled", "isCommitment": True}}],
                requests,
            ),
        )
    )

    sdk.payout.create(
        {
            "type": "Milestone",
            "chain": "base",
            "name": "Website build",
            "payoutCurrency": "usdc",
            "metadata": {"milestones": [{"name": "Design approval", "amount": 500, "dueDate": "2026-07-01T00:00:00.000Z", "status": "pending"}]},
        }
    )

    payload = json.loads(requests[0]["payload"])
    assert payload["paymentType"] == "Scheduled"
    assert payload["isCommitment"] is True
    assert payload["metadata"]["commitmentType"] == "milestone"


def test_compute_escrow_funding_digests_match_packed_encoding():
    escrow_batch_hash = "0x" + "11" * 32
    withdrawal_wallet = "0x0000000000000000000000000000000000000003"
    merkle_root = "0x" + "22" * 32

    expected_funding = to_hex(keccak(encode_packed(["bytes32", "address"], [bytes.fromhex(escrow_batch_hash[2:]), withdrawal_wallet])))
    expected_scheduled = to_hex(keccak(encode_packed(["bytes32", "bytes32"], [bytes.fromhex(escrow_batch_hash[2:]), bytes.fromhex(merkle_root[2:])])))

    assert computeEscrowFundingDigest({"escrowBatchHash": escrow_batch_hash, "withdrawalWallet": withdrawal_wallet}) == expected_funding
    assert computeEscrowScheduledFundingDigest({"escrowBatchHash": escrow_batch_hash, "merkleRoot": merkle_root}) == expected_scheduled


def test_finalize_accepts_message_signing_function_for_instant_payouts():
    requests = []
    responses = [
        {
            "meta": {"statusCode": 200, "success": True},
            "data": {
                "id": "120bdabb-5790-415c-ae75-c2fca1cc5232",
                "chain": "base",
                "paymentType": "Instant",
                "complianceMode": "Open",
                "nonce": "0x1234",
                "app": {"clientId": "app_test"},
                "payments": [
                    {
                        "receiver": "0x0000000000000000000000000000000000000001",
                        "amount": "1",
                        "token": "0x0000000000000000000000000000000000000002",
                        "decimals": 6,
                        "memo": "",
                    }
                ],
            },
        },
        {
            "meta": {"statusCode": 200, "success": True},
            "data": {
                "id": "120bdabb-5790-415c-ae75-c2fca1cc5232",
                "chain": "base",
                "paymentType": "Instant",
                "batchDataHash": "0xabc",
            },
        },
    ]
    sdk = PviumSdk.init(PviumSdkConfig(baseUrl="http://localhost:4005/v1", fetchFn=make_fetch(responses, requests)))
    signed_messages = []

    sdk.payout.finalize(
        "120bdabb-5790-415c-ae75-c2fca1cc5232",
        lambda message: signed_messages.append(message) or "0xsigned",
        {"signerAddress": "0x0000000000000000000000000000000000000003", "timestamp": 123},
    )

    assert len(signed_messages) == 1
    assert signed_messages[0].startswith("PVIUM_SIGNED_BATCH:app_test:")
    assert requests[1]["method"] == "PATCH"
    payload = json.loads(requests[1]["payload"])
    assert payload["signer"] == "0x0000000000000000000000000000000000000003"
    assert payload["batchSignature"] == "123:0x0000000000000000000000000000000000000003:0xsigned"


def test_finalize_chain_override_can_skip_scheduled_funding_signature_for_solana():
    requests = []
    responses = [
        {
            "meta": {"statusCode": 200, "success": True},
            "data": {
                "id": "120bdabb-5790-415c-ae75-c2fca1cc5232",
                "chain": "base",
                "paymentType": "Scheduled",
                "complianceMode": "Open",
                "metadata": {
                    "payoutCurrency": "0x0000000000000000000000000000000000000002",
                    "gracePeriod": 0,
                    "disapprovalDeadline": 0,
                    "scheduledDate": 0,
                },
                "app": {"clientId": "app_test"},
                "payments": [
                    {
                        "receiver": "0x0000000000000000000000000000000000000001",
                        "amount": "1",
                        "token": "0x0000000000000000000000000000000000000002",
                        "decimals": 6,
                        "memo": "",
                    }
                ],
            },
        },
        {
            "meta": {"statusCode": 200, "success": True},
            "data": {"id": "batch_1", "chain": "solana", "paymentType": "Scheduled", "merkleRoot": "0xabc"},
        },
    ]
    sdk = PviumSdk.init(PviumSdkConfig(baseUrl="http://localhost:4005/v1", fetchFn=make_fetch(responses, requests)))

    sdk.payout.finalize(
        "120bdabb-5790-415c-ae75-c2fca1cc5232",
        lambda _message: "signed",
        {"chain": "solana", "chainId": 1, "signerAddress": "0x0000000000000000000000000000000000000003", "timestamp": 123},
    )

    payload = json.loads(requests[1]["payload"])
    assert "fundingSignature" not in payload
    assert payload["batchSignature"].endswith(":signed")


def test_finalize_supports_separate_finalize_and_funding_signers():
    requests = []
    calls = []
    responses = [
        {
            "meta": {"statusCode": 200, "success": True},
            "data": {
                "id": "120bdabb-5790-415c-ae75-c2fca1cc5232",
                "chain": "base",
                "paymentType": "Scheduled",
                "complianceMode": "Open",
                "metadata": {
                    "payoutCurrency": "0x0000000000000000000000000000000000000002",
                    "gracePeriod": 0,
                    "disapprovalDeadline": 0,
                    "scheduledDate": 0,
                },
                "app": {"clientId": "app_test"},
                "payments": [
                    {
                        "receiver": "0x0000000000000000000000000000000000000001",
                        "amount": "1",
                        "token": "0x0000000000000000000000000000000000000002",
                        "decimals": 6,
                        "memo": "",
                    }
                ],
            },
        },
        {
            "meta": {"statusCode": 200, "success": True},
            "data": {"id": "120bdabb-5790-415c-ae75-c2fca1cc5232", "chain": "base", "paymentType": "Scheduled", "merkleRoot": "0xabc"},
        },
    ]
    sdk = PviumSdk.init(PviumSdkConfig(baseUrl="http://localhost:4005/v1", fetchFn=make_fetch(responses, requests)))

    sdk.payout.finalize(
        "120bdabb-5790-415c-ae75-c2fca1cc5232",
        {
            "chain": "ethereum",
            "signerAddress": "0x0000000000000000000000000000000000000003",
            "signMessage": lambda _message: (_ for _ in ()).throw(RuntimeError("fallback signMessage should not be called")),
            "signFinalize": lambda message: calls.append(f"finalize:{message}") or "finalize-signature",
            "signFunding": lambda digest: calls.append(f"funding:{digest}") or "funding-signature",
        },
        {"chain": "base", "timestamp": 123},
    )

    assert len(calls) == 2
    assert calls[0].startswith("finalize:PVIUM_SIGNED_SCHEDULE:")
    assert calls[1].startswith("funding:0x")
    payload = json.loads(requests[1]["payload"])
    assert payload["batchSignature"].endswith(":finalize-signature")
    assert payload["fundingSignature"] == "funding-signature"


def test_add_payments_creates_and_signs_escrow_child_scheduled_payout_with_private_key():
    requests = []
    private_key = "0x59c6995e998f97a5a0044976f0d7f3f6f8f53f6a2046baf4f01cb4f1f6bcb58f"
    escrow_batch = {
        "id": "7a6ca76d-77f7-4c0e-9da9-c64f1cb18a1f",
        "chain": "base",
        "paymentType": "Escrow",
        "status": "funded",
        "complianceMode": "Open",
        "name": "Creator escrow",
        "batchHash": "0x" + "11" * 32,
        "metadata": {"payoutCurrency": "0x0000000000000000000000000000000000000002"},
        "app": {"clientId": "app_test"},
    }
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch(
                [
                    {
                        "meta": {"statusCode": 201, "success": True},
                        "data": {
                            "id": "22222222-2222-4222-8222-222222222222",
                            "chain": "base",
                            "paymentType": "Scheduled",
                            "escrowBatch": escrow_batch["id"],
                            "merkleRoot": "0x" + "aa" * 32,
                        },
                    }
                ],
                requests,
            ),
        )
    )

    escrow_intent = PayoutIntent(sdk.payout, {"statusCode": 200, "success": True}, escrow_batch)
    escrow_intent.addPayments(
        {
            "payments": [
                {
                    "receiver": "0x0000000000000000000000000000000000000001",
                    "amount": "25",
                    "decimals": 6,
                    "memo": "escrow work",
                }
            ],
            "signer": private_key,
            "finalizeOptions": {"id": "22222222-2222-4222-8222-222222222222", "timestamp": 1777487451, "claimDate": 1777488000},
        },
    )

    assert len(requests) == 1
    assert requests[0]["url"] == "http://localhost:4005/v1/batch-payments"
    assert requests[0]["method"] == "POST"
    payload = json.loads(requests[0]["payload"])
    assert payload["id"] == "22222222-2222-4222-8222-222222222222"
    assert payload["paymentType"] == "Scheduled"
    assert payload["escrowBatch"] == escrow_batch["id"]
    assert re.match(r"^1777487451:0x[0-9a-f]{40}:0x", payload["batchSignature"], re.I)
    assert re.match(r"^0x[0-9a-f]+$", payload["fundingSignature"], re.I)
    assert re.match(r"^0x[0-9a-f]{64}$", payload["batchHash"], re.I)
    assert re.match(r"^0x[0-9a-f]{64}$", payload["batchDataHash"], re.I)
    assert re.match(r"^0x[0-9a-f]{64}$", payload["merkleRoot"], re.I)
    assert payload["proofs"][0]["receiver"] == "0x0000000000000000000000000000000000000001"
    assert payload["metadata"]["escrowBatch"] == escrow_batch["id"]
    assert payload["metadata"]["escrowBatchHash"] == escrow_batch["batchHash"]
    assert payload["metadata"]["scheduledDate"] == 1777488000
    assert payload["payments"][0]["claimDate"] == 1777488000
    assert payload["payments"][0]["token"] == "0x0000000000000000000000000000000000000002"


def test_add_payments_rejects_escrow_payouts_without_signer():
    sdk = PviumSdk.init(PviumSdkConfig(baseUrl="http://localhost:4005/v1", fetchFn=make_fetch([], [])))

    try:
        sdk.payout.addPayments(
            {
                "id": "7a6ca76d-77f7-4c0e-9da9-c64f1cb18a1f",
                "chain": "base",
                "paymentType": "Escrow",
                "status": "funded",
                "batchHash": "0x" + "11" * 32,
                "metadata": {"payoutCurrency": "0x0000000000000000000000000000000000000002"},
                "app": {"clientId": "app_test"},
            },
            [{"receiver": "0x0000000000000000000000000000000000000001", "amount": "25", "decimals": 6}],
        )
    except RuntimeError as err:
        assert "signer or private key is required" in str(err)
    else:
        raise AssertionError("expected signer error")


def test_scheduled_payout_merkle_promotes_odd_leaf_like_node():
    base_payments = [
        {
            "receiver": "0x0000000000000000000000000000000000000001",
            "amount": "1",
            "token": "0x0000000000000000000000000000000000000002",
            "decimals": 6,
            "memo": "a",
        },
        {
            "receiver": "0x0000000000000000000000000000000000000003",
            "amount": "2",
            "token": "0x0000000000000000000000000000000000000002",
            "decimals": 6,
            "memo": "b",
        },
        {
            "receiver": "0x0000000000000000000000000000000000000004",
            "amount": "3",
            "token": "0x0000000000000000000000000000000000000002",
            "decimals": 6,
            "memo": "c",
        },
    ]
    expected = [
        {
            "count": 1,
            "merkleRoot": "0xc40e60ab1b114ed2eb4bbe156b73023238565bf37b45766c3c922b8089d1d2e9",
            "proofs": [[]],
        },
        {
            "count": 2,
            "merkleRoot": "0x7ade83cae70b4278f73144e9a99f77f7deeed719e4778245b9f3db71f6ae02b7",
            "proofs": [
                ["0xdeaa9357ef2c59449293ed3ba3060e6aa9cf4bf4812ecc96f2b3a6500744f05b"],
                ["0xc40e60ab1b114ed2eb4bbe156b73023238565bf37b45766c3c922b8089d1d2e9"],
            ],
        },
        {
            "count": 3,
            "merkleRoot": "0xf9eb35f3a7cf94793c4f0d440f9c510828d516e6ac308141819ab7265754a8d6",
            "proofs": [
                [
                    "0xdeaa9357ef2c59449293ed3ba3060e6aa9cf4bf4812ecc96f2b3a6500744f05b",
                    "0x54acdb6206c4f539cbcda16b5132e71254f0ae9126b4efaa442f5c61659d544c",
                ],
                [
                    "0xc40e60ab1b114ed2eb4bbe156b73023238565bf37b45766c3c922b8089d1d2e9",
                    "0x54acdb6206c4f539cbcda16b5132e71254f0ae9126b4efaa442f5c61659d544c",
                ],
                ["0x7ade83cae70b4278f73144e9a99f77f7deeed719e4778245b9f3db71f6ae02b7"],
            ],
        },
    ]

    for item in expected:
        requests = []
        responses = [
            {
                "meta": {"statusCode": 200, "success": True},
                "data": {
                    "id": "120bdabb-5790-415c-ae75-c2fca1cc5232",
                    "chain": "base",
                    "paymentType": "Scheduled",
                    "complianceMode": "Open",
                    "metadata": {
                        "payoutCurrency": "0x0000000000000000000000000000000000000002",
                        "gracePeriod": 0,
                        "disapprovalDeadline": 0,
                        "scheduledDate": 0,
                    },
                    "app": {"clientId": "app_test"},
                    "payments": base_payments[: item["count"]],
                },
            },
            {"meta": {"statusCode": 200, "success": True}, "data": {"id": "x", "paymentType": "Scheduled"}},
        ]
        sdk = PviumSdk.init(PviumSdkConfig(baseUrl="http://localhost:4005/v1", fetchFn=make_fetch(responses, requests)))

        sdk.payout.finalize(
            "120bdabb-5790-415c-ae75-c2fca1cc5232",
            {
                "signerAddress": "0x0000000000000000000000000000000000000005",
                "signFinalize": lambda _message: "finalize",
                "signFunding": lambda _digest: "funding",
            },
            {"chain": "base", "timestamp": 123, "claimDate": 1777488000},
        )

        payload = json.loads(requests[1]["payload"])
        assert payload["merkleRoot"] == item["merkleRoot"]
        assert [proof["proof"] for proof in payload["proofs"]] == item["proofs"]


def test_scheduled_payout_finalization_signatures_match_node_parity_values():
    fixture = load_parity_fixture()
    requests = []
    responses = [fixture["getResponse"], fixture["patchResponse"]]
    sdk = PviumSdk.init(PviumSdkConfig(baseUrl="http://localhost:4005/v1", fetchFn=make_fetch(responses, requests)))

    result = sdk.payout.finalize(
        fixture["payoutId"],
        fixture["privateKey"],
        fixture["options"],
    )

    payload = json.loads(requests[1]["payload"])
    assert payload == fixture["expectedPatchPayload"]
    assert result["data"]["payout"] == fixture["expectedResult"]["payout"]
    assert result["data"]["fundingUrl"] == fixture["expectedResult"]["fundingUrl"]
    assert result["data"]["batchDataHash"] == fixture["expectedResult"]["batchDataHash"]
    assert result["data"]["batchHash"] == fixture["expectedResult"]["batchHash"]
    assert result["data"]["merkleRoot"] == fixture["expectedResult"]["merkleRoot"]
