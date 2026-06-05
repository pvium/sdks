import json
from urllib.parse import parse_qs, urlparse

from eth_account import Account
from eth_account.messages import encode_defunct

from pvium_sdk import PviumSdk, PviumSdkConfig, generateBatchInviteMerkleDataV2


TEST_PRIVATE_KEY = "0x59c6995e998f97a5a0044976f0d7f3f6f8f53f6a2046baf4f01cb4f1f6bcb58f"
TEST_ADDRESS = Account.from_key(TEST_PRIVATE_KEY).address


def test_create_and_sign_oauth_invite_bundle_with_evm_key():
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            consentHost="http://localhost:3000",
            clientId="app_test",
            apiKey="pk_test_dummy",
        )
    )

    bundle = sdk.invites.createBundle(
        {
            "identities": [{"type": "email", "value": "Test.User@example.com"}],
            "scopes": ["read:ethereum_wallet", "read:user"],
            "chain": "ethereum",
        }
    )

    signed = sdk.invites.signBundle(bundle, {"chain": "ethereum", "privateKey": TEST_PRIVATE_KEY})

    assert signed["clientId"] == "app_test"
    assert len(signed["invites"]) == 1
    assert signed["root"]["signatureType"] == "evm-personal-sign"
    assert signed["root"]["signerAddress"] == TEST_ADDRESS
    assert signed["scopes"] == ["read:ethereum_wallet", "read:user"]

    recovered = Account.recover_message(
        encode_defunct(text=signed["root"]["signatureMessage"]),
        signature=signed["root"]["signature"],
    )
    assert recovered == TEST_ADDRESS

    invite = signed["invites"][0]
    assert invite["identityType"] == "email"
    assert invite["identityValue"] == "test.user@example.com"
    assert invite["appClientId"] == "app_test"
    assert invite["leafVersion"] == "2"
    assert isinstance(invite["secretHash"], str)
    assert len(invite["secretHash"]) == 64
    assert len(invite["proof"]) == 0

    invite_url = urlparse(invite["inviteLink"])
    query = parse_qs(invite_url.query)
    assert f"{invite_url.scheme}://{invite_url.netloc}" == "http://localhost:3000"
    assert invite_url.path == "/oauth2/authorize"
    assert query["client_id"] == ["app_test"]
    assert query["response_type"] == ["code"]
    assert query["scope"] == ["read:ethereum_wallet read:user"]
    assert query["invite_nonce"] == [invite["inviteNonce"]]
    assert query["invite_secret"] == [invite["inviteSecret"]]
    assert query["identity_type"] == ["email"]
    assert query["identity_hint"] == ["test.user@example.com"]


def test_creates_batch_invite_bundle_links_with_explicit_batch_id_and_custom_state():
    requests = []

    def fetch(method, url, headers, payload, timeout):
        requests.append({"method": method, "url": url, "headers": headers, "payload": payload})
        return 201, {"content-type": "application/json"}, json.dumps({"ok": True})

    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            consentHost="http://localhost:3000",
            clientId="app_test",
            apiKey="pk_test_dummy",
            fetchFn=fetch,
        )
    )

    bundle = sdk.invites.createBundle(
        {
            "identities": [{"type": "email", "value": "Batch.User@example.com"}],
            "scopes": ["read:user", "read:ethereum_wallet"],
            "chain": "ethereum",
            "batchInvite": {"batchId": "batch_123", "stateParams": {"source": "sdk-test"}},
            "stateParams": {"returnTo": "/admin/bulk-payments/batch_123"},
        }
    )

    signed = sdk.invites.signBundle(bundle, {"chain": "ethereum", "privateKey": TEST_PRIVATE_KEY})

    assert signed["batchId"] == "batch_123"
    assert signed["batchInvite"]["batchId"] == "batch_123"

    invite_query = parse_qs(urlparse(signed["invites"][0]["inviteLink"]).query)
    assert invite_query["batchId"] == ["batch_123"]
    state = parse_qs(invite_query["state"][0])
    assert state["batchId"] == ["batch_123"]
    assert state["source"] == ["sdk-test"]
    assert state["returnTo"] == ["/admin/bulk-payments/batch_123"]

    group_query = parse_qs(urlparse(signed["groupInviteLink"]).query)
    assert group_query["batchId"] == ["batch_123"]

    sdk.invites.commitBundle(signed)

    assert len(requests) == 1
    assert requests[0]["url"] == "http://localhost:4005/v1/batch-payments/batch_123/invites"


def test_supports_separate_master_secret_and_invite_root_signers():
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            consentHost="http://localhost:3000",
            clientId="app_test",
            apiKey="pk_test_dummy",
        )
    )
    account = Account.from_key(TEST_PRIVATE_KEY)
    calls = []

    bundle = sdk.invites.createBundle(
        {
            "identities": [{"type": "email", "value": "Split.Signer@example.com"}],
            "scopes": ["read:user", "read:ethereum_wallet"],
            "chain": "ethereum",
        }
    )

    def sign_master(message):
        calls.append(f"master:{message}")
        return account.sign_message(encode_defunct(text=message)).signature.hex()

    def sign_root(message):
        calls.append(f"root:{message}")
        return account.sign_message(encode_defunct(text=message)).signature.hex()

    signed = sdk.invites.signBundle(
        bundle,
        {
            "chain": "ethereum",
            "signerAddress": account.address,
            "signMessage": lambda _message: (_ for _ in ()).throw(RuntimeError("fallback signMessage should not be called")),
            "signMasterSecret": sign_master,
            "signInviteRoot": sign_root,
        },
    )

    assert len(calls) == 2
    assert calls[0].startswith("master:PVIUM_INVITE_SECRET_V2:")
    assert calls[1].startswith("root:PVIUM_INVITE_ROOT_V2")
    assert signed["root"]["signerAddress"] == account.address
    recovered = Account.recover_message(encode_defunct(text=signed["root"]["signatureMessage"]), signature=signed["root"]["signature"])
    assert recovered == account.address


def test_invite_merkle_promotes_odd_leaf_like_node():
    merkle = generateBatchInviteMerkleDataV2(
        {
            "appClientId": "app_test",
            "batchId": "batch_odd",
            "scopes": ["read:user", "read:ethereum_wallet"],
            "createdAt": 1700000000,
            "rootNonce": "abcdef1234567890abcdef1234567890",
            "invites": [
                {
                    "identityType": "email",
                    "identityValue": "a@example.com",
                    "inviteNonce": "11111111111111111111111111111111",
                    "inviteSecret": "2" * 64,
                    "expiresAt": 1716250000,
                },
                {
                    "identityType": "email",
                    "identityValue": "b@example.com",
                    "inviteNonce": "33333333333333333333333333333333",
                    "inviteSecret": "4" * 64,
                    "expiresAt": 1716250000,
                },
                {
                    "identityType": "email",
                    "identityValue": "c@example.com",
                    "inviteNonce": "55555555555555555555555555555555",
                    "inviteSecret": "6" * 64,
                    "expiresAt": 1716250000,
                },
            ],
        }
    )

    assert merkle["root"] == "0xaab8dc5fb2ea9d657df953d819143db57ee2216739596a8573d14855cf37f314"
    assert merkle["invites"][2]["proof"] == ["0x0503e00299cf7719a692ed9dd737e2c61ab7ca83c2419db9793cfab2b6626cc1"]
