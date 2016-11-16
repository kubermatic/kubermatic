#!/usr/bin/env python
import unittest

import mock
from ct.cert_analysis import base_check_test
from ct.cert_analysis import crl_pointers
from ct.crypto import cert

CRYPTO_TEST_DATA_DIR = "ct/crypto/testdata/"
CERT_WITH_CRL = cert.Certificate.from_pem_file(CRYPTO_TEST_DATA_DIR +
                                                "aia.pem")
CERT_WITHOUT_CRL = cert.Certificate.from_pem_file(CRYPTO_TEST_DATA_DIR +
                                                   "promise_com.pem")

class CrlPointersTest(base_check_test.BaseCheckTest):
    def test_crl_existence_exist(self):
        check = crl_pointers.CheckCrlExistence()
        result = check.check(CERT_WITH_CRL)
        self.assertIsNone(result)

    def test_crl_existence_doesnt_exist(self):
        check = crl_pointers.CheckCrlExistence()
        result = check.check(CERT_WITHOUT_CRL)
        self.assertObservationIn(crl_pointers.LackOfCrl(), result)

    def test_crl_extension_corrupt(self):
        certificate = mock.MagicMock()
        certificate.crl_distribution_points = mock.Mock(
                side_effect=cert.CertificateError("Corrupt or unrecognized..."))
        check = crl_pointers.CheckCorruptOrMultipleCrlExtension()
        result = check.check(certificate)
        self.assertObservationIn(crl_pointers.CorruptCrlExtension(), result)

    def test_crl_extension_multiple(self):
        certificate = mock.MagicMock()
        certificate.crl_distribution_points = mock.Mock(
                side_effect=cert.CertificateError("Multiple extension values"))
        check = crl_pointers.CheckCorruptOrMultipleCrlExtension()
        result = check.check(certificate)
        self.assertObservationIn(crl_pointers.MultipleCrlExtensions(), result)


if __name__ == '__main__':
    unittest.main()
