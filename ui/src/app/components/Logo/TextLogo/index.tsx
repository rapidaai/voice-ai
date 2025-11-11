export function TextLogo(props: { isBeta: boolean; isDev?: boolean }) {
  return (
    <>
      {/* <span className="self-center text-xl font-semibold sm:text-2xl whitespace-nowrap dark:text-white">
        rapida.ai
      </span> */}
      {props.isBeta ? (
        <span className="align-text-bottom from-rose-600 via-pink-600 to-blue-600 bg-linear-to-r bg-clip-text text-transparent">
          beta
        </span>
      ) : props.isDev ? (
        <span className="align-text-bottom from-rose-600 via-pink-600 to-blue-600 bg-linear-to-r bg-clip-text text-transparent">
          dev
        </span>
      ) : (
        <></>
      )}
    </>
  );
}
