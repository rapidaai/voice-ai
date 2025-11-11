import React, { useState } from 'react';
import { Helmet } from '@/app/components/Helmet';
import { DescriptiveHeading } from '@/app/components/Heading/DescriptiveHeading';

/**
 *
 * @returns
 */
export function BillingPage() {
  /**
   *
   */
  const [paymentMethod, setPaymentMethod] = useState<string>(
    'Mastercard ending 9282',
  );
  const [billingInterval, setBillingInterval] = useState<string>('Annually');
  const [taxInformation, setTaxInformation] = useState<string>('UK849700927');
  const [billingAddress, setBillingAddress] = useState<string>(
    '34 Savoy Street, London, UK, 24E8X',
  );
  const [email, setEmail] = useState<string>('hello@cruip.com');

  const updateBillingInformation = () => {
    // UpdateBillingInformation(
    //   {
    //     paymentMethod,
    //     taxInformation,
    //     address: billingAddress,
    //     email,
    //     billingInterval,
    //   },
    //   (err: ServiceError | null, auth: BaseResponse | null) => {
    //     if (err) {
    //       // setError('unable to process your request. please try again later.');
    //       return;
    //     }
    //     if (auth?.getSuccess()) {
    //       console.log('Updated billing address');
    //     } else {
    //       // setError('please provide valid credentials to signin into account.');
    //       return;
    //     }
    //   },
    // );
  };

  /**
   *
   * invoices
   */

  const [invoices, setInvoices] = useState([]);
  return (
    <>
      <Helmet title="Plan Details & Billing"></Helmet>

      <div className="space-y-8 my-10">
        <div>
          <DescriptiveHeading heading="Receipt Information" />
        </div>

        {/* Billing Information */}
        <section>
          <ul className="mt-5">
            <li className="border-gray-200 dark:border-gray-800 py-3 border-b items-center flex justify-between">
              {/* Left */}
              <div className="text-sm font-medium">Payment Method</div>
              {/* Right */}
              <div className="text-sm dark:text-gray-400ml-4">
                <span className="mr-3">Not available</span>
                <a className="text-blue-500 font-medium underline" href="#0">
                  Edit
                </a>
              </div>
            </li>
            <li className="border-gray-200 dark:border-gray-800 py-3 border-b items-center flex justify-between">
              {/* Left */}
              <div className="text-sm font-medium">Billing Interval</div>
              {/* Right */}
              <div className="text-sm dark:text-gray-400ml-4">
                <span className="mr-3">Not available</span>
                <a className="text-blue-500 font-medium underline" href="#0">
                  Edit
                </a>
              </div>
            </li>
            <li className="border-gray-200 dark:border-gray-800 py-3 border-b items-center flex justify-between">
              {/* Left */}
              <div className="text-sm font-medium">VAT/GST Number</div>
              {/* Right */}
              <div className="text-sm dark:text-gray-400ml-4">
                <span className="mr-3">Not available</span>
                <a className="text-blue-500 font-medium underline" href="#0">
                  Edit
                </a>
              </div>
            </li>
            <li className="border-gray-200 dark:border-gray-800 py-3 border-b items-center flex justify-between">
              {/* Left */}
              <div className="text-sm font-medium">Your Address</div>
              {/* Right */}
              <div className="text-sm dark:text-gray-400ml-4">
                <span className="mr-3">Not available</span>
                <a className="text-blue-500 font-medium underline" href="#0">
                  Edit
                </a>
              </div>
            </li>
            <li className="border-gray-200 dark:border-gray-800 py-3 border-b items-center flex justify-between">
              {/* Left */}
              <div className="text-sm font-medium">Billing Address</div>
              {/* Right */}
              <div className="text-sm dark:text-gray-400ml-4">
                <span className="mr-3">Not available</span>
                <a className="text-blue-500 font-medium underline" href="#0">
                  Edit
                </a>
              </div>
            </li>
          </ul>
        </section>
        {/* Invoices */}
        <section className="overflow-x-auto border-gray-200 dark:border-gray-700 border rounded-[2px]">
          <div className="px-5 py-3 bg-white dark:bg-gray-800 flex justify-between items-center">
            <DescriptiveHeading
              heading="Invoices"
              subheading="Your past invoices and payments."
            />
          </div>
          {invoices.length > 0 ? (
            <table className="table-auto text-left w-full">
              <thead className="">
                <tr className="text-gray-500 dark:text-gray-400 border-gray-200 dark:border-gray-700 dark:bg-gray-900/20 font-semibold bg-gray-50 uppercase border-b border-t text-xs">
                  <th className="uppercase font-semibold text-xs py-2 px-5">
                    #Id
                  </th>
                  <th className="uppercase font-semibold text-xs py-2 px-5">
                    Date
                  </th>
                  <th className="uppercase font-semibold text-xs py-2 px-5">
                    Plan
                  </th>
                  <th className="uppercase font-semibold text-xs py-2 px-5">
                    Amount
                  </th>
                  <th></th>
                </tr>
              </thead>
              <tbody className="space-y-1">
                {invoices.map((ic, idx) => {
                  return <InvoiceRecord key={idx} />;
                })}
              </tbody>
            </table>
          ) : (
            <p className="px-2 md:px-5 py-3">
              You have not made any payments in past 6 months.
            </p>
          )}
        </section>
      </div>
    </>
  );
}

function InvoiceRecord() {
  return (
    <tr className="border-b border-gray-200 dark:border-gray-800">
      <td className="py-2 text-sm px-5">1961</td>
      <td className="py-2 text-sm px-5">1961</td>
      <td className="py-2 text-sm px-5">Basic Plan - Annualy</td>
      <td className="py-2 text-sm px-5">$429</td>
      <td className="py-2 text-sm px-5">
        <div className="flex items-center space-x-2 justify-end">
          <a className="text-blue-500 font-medium underline" href="#0">
            View
          </a>
          <span className="border-r h-2 border-gray-300 dark:border-gray-500" />
          <a className="text-blue-500 font-medium underline" href="#0">
            Download
          </a>
        </div>
      </td>
    </tr>
  );
}
