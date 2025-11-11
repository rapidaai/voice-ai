export function DescriptiveHeading(props: {
  heading: string;
  info?: string;
  subheading?: string;
}) {
  return (
    <div className="flex flex-col py-2">
      <h1 className="dark:text-gray-100 text-xl">
        {props.heading}
        {props.info && <small className="text-base ml-2">({props.info})</small>}
      </h1>
      <h3 className="text-base mt-2">{props.subheading}</h3>
    </div>
  );
}
