#!/usr/bin/perl

use strict;
use CGI;
use Archive::Zip;   # imports
use IO::Scalar;

my $obj = Archive::Zip->new();
my %frmVals = ();
my $ret = &ReadForm(\%frmVals);


if ($frmVals{'pid'}) {

	my $pid = $frmVals{'pid'};
	my $file;
	
	if(-e "/tmp/$pid.tre") {
    		$obj->addFile("/tmp/$pid.tre","$pid.tre");   # add files
	}
	if(-e "/tmp/$pid.txt") {
		$obj->addFile("/tmp/$pid.txt","$pid.txt");
	}	
	if(-e "/tmp/$pid.invalid_names.txt") {
		$obj->addFile("/tmp/$pid.invalid_names.txt","$pid.invalid_names.txt");
	}	

	my $memory_file = '';   #scalar as a file
	my $memfile_fh = IO::Scalar->new(\$memory_file); #filehandle to the scalar

	# write to the scalar $memory_file
	my $status = $obj->writeToFileHandle($memfile_fh);
	$memfile_fh->close;

	#print with apache
	#print("Content-Type:application/zip");
	print "Content-Type:application/x-download\n";
         print "Content-Disposition:attachment;filename=$pid.zip\n\n";

	print($memory_file);    #the content of a file-in-a-scalar
}

#==============================================================
# subroutine "ReadForm" parses the data string that comes in
# from a client web browser by way of Apache and stores it in 
# a hash called %variable, where the keys are the names of the 
# form items, and the values are their respective contents
sub ReadForm {
	my ($varRef) = @_;
	my ($i, $key, $val, $variable, @list);
	
	if (&MethGet) {
		$variable = $ENV{'QUERY_STRING'};
	} elsif (&MethPost) {
		read(STDIN, $variable, $ENV{'CONTENT_LENGTH'});
	}
	
	@list = split(/[&;]/,$variable);
	
	foreach $i (0 .. $#list) {
		$list[$i] =~ s/\+/ /g;
		($key, $val) = split(/=/,$list[$i],2);
		$key =~ s/%(..)/pack("c",hex($1))/ge;
		$val =~ s/%(..)/pack("c",hex($1))/ge;
		$$varRef{$key} .= "\0" if (defined($$varRef{$key}));
		$$varRef{$key} .= $val;
	}
	return scalar($#list + 1);
}
sub MethGet { return ($ENV{'REQUEST_METHOD'} eq "GET") };
sub MethPost { return ($ENV{'REQUEST_METHOD'} eq "POST") };


