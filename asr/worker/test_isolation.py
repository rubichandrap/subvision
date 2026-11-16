import unittest

import numpy as np

from isolation import apply_isolation, enabled_from_env


class EnabledFromEnv(unittest.TestCase):
    def test_off_by_default(self):
        for raw in (None, "", "0", "false", "No", "off"):
            self.assertFalse(enabled_from_env(raw), raw)

    def test_on(self):
        for raw in ("1", "true", "YES", " on "):
            self.assertTrue(enabled_from_env(raw), raw)

    def test_garbage_is_loud(self):
        with self.assertRaises(ValueError):
            enabled_from_env("maybe")


class ApplyIsolation(unittest.TestCase):
    def test_returns_isolated_float32_of_same_length(self):
        samples = np.ones(1600, dtype=np.float32)
        out = apply_isolation(samples, 16000, lambda s, sr: s * 0.5)
        self.assertEqual(out.dtype, np.float32)
        self.assertEqual(out.shape, samples.shape)
        self.assertAlmostEqual(float(out[0]), 0.5)

    def test_length_change_is_rejected(self):
        samples = np.zeros(1600, dtype=np.float32)
        with self.assertRaises(ValueError):
            apply_isolation(samples, 16000, lambda s, sr: s[:-1])


if __name__ == "__main__":
    unittest.main()
