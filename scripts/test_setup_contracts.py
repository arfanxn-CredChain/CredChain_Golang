import importlib.util
import unittest
from pathlib import Path

# setup-contracts.py has a hyphen, so load it by path rather than import.
MODULE_PATH = Path(__file__).with_name("setup-contracts.py")
spec = importlib.util.spec_from_file_location("setup_contracts", MODULE_PATH)
sc = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sc)


class ContractsExistTest(unittest.TestCase):
    def test_skips_when_both_have_code(self):
        addrs = {"authority": "0xAAA", "registry": "0xBBB"}
        self.assertTrue(sc.contracts_exist(addrs, get_code_fn=lambda a: "0x60806040"))

    def test_deploys_when_code_missing(self):
        addrs = {"authority": "0xAAA", "registry": "0xBBB"}
        self.assertFalse(sc.contracts_exist(addrs, get_code_fn=lambda a: "0x"))

    def test_deploys_when_address_absent(self):
        addrs = {"authority": "", "registry": "0xBBB"}
        self.assertFalse(sc.contracts_exist(addrs, get_code_fn=lambda a: "0x60806040"))

    def test_read_env_addresses_parses_both(self):
        env = Path(__file__).with_name("_tmp_env_for_test")
        env.write_text("FOO=bar\nAUTHORITY_CONTRACT=0x99\nREGISTRY_CONTRACT=0x0E\n")
        try:
            addrs = sc.read_env_addresses(env)
            self.assertEqual(addrs["authority"], "0x99")
            self.assertEqual(addrs["registry"], "0x0E")
        finally:
            env.unlink()


if __name__ == "__main__":
    unittest.main()
